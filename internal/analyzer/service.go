package analyzer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	parserpkg "github.com/codeatlas/codeatlas/internal/parser"
	"github.com/codeatlas/codeatlas/internal/repository"
	"github.com/codeatlas/codeatlas/internal/store"
)

type Service struct {
	store  *store.Store
	logger *slog.Logger
	cache  sync.Map
}

func New(database *store.Store, logger *slog.Logger) *Service {
	return &Service{store: database, logger: logger}
}

func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(650 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run, err := s.store.ClaimRun(ctx)
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			if err != nil {
				s.logger.Error("claim analysis job", "error", err)
				continue
			}
			if err := s.analyze(ctx, run.ID, run.ProjectID); err != nil {
				s.logger.Error("analysis failed", "project_id", run.ProjectID, "run_id", run.ID, "error", err)
			}
		}
	}
}

func (s *Service) analyze(ctx context.Context, runID, projectID string) (finalErr error) {
	project, err := s.store.Project(ctx, projectID)
	if err != nil {
		return err
	}
	defer func() {
		if finalErr != nil {
			safe := sanitizeError(finalErr, project.RootPath)
			_ = s.store.FailRun(context.Background(), runID, projectID, safe)
		}
	}()
	_ = s.store.UpdateRun(ctx, runID, "acquiring", 0, 0, 0, 0, "Preparing repository")
	commit, err := repository.Acquire(ctx, project)
	if err != nil {
		return err
	}
	if commit != "" {
		_ = s.store.UpdateProjectCommit(ctx, projectID, commit)
	}
	_ = s.store.UpdateRun(ctx, runID, "discovering", 0, 0, 0, 0, "Inventorying local files")
	discovered, err := repository.Discover(ctx, projectID, runID, project.RootPath)
	if err != nil {
		return err
	}
	sourceCount := 0
	for _, item := range discovered {
		if item.Record.Classification == "source" {
			sourceCount++
		}
	}
	if sourceCount > 10000 {
		return fmt.Errorf("repository has %d relevant source files; the MVP limit is 10000", sourceCount)
	}
	_ = s.store.UpdateRun(ctx, runID, "parsing", 0, sourceCount, 0, 0, fmt.Sprintf("Parsing %d source files", sourceCount))
	parsed, err := s.parseFiles(ctx, runID, discovered, sourceCount)
	if err != nil {
		return err
	}
	entitiesScanned, edgesScanned := seedCounts(parsed)
	_ = s.store.UpdateRun(ctx, runID, "resolving", sourceCount, sourceCount, entitiesScanned, edgesScanned, "Resolving symbols and relationships")
	graph := buildGraph(projectID, runID, discovered, parsed)
	_ = s.store.UpdateRun(ctx, runID, "persisting", sourceCount, sourceCount, len(graph.Entities), len(graph.Relationships), "Persisting derived metadata")
	if err := s.store.ReplaceRunData(ctx, runID, graph.Files, graph.Entities, graph.Relationships); err != nil {
		return err
	}
	if err := s.store.PromoteRun(ctx, runID, projectID); err != nil {
		return err
	}
	s.logger.Info("analysis complete", "project_id", projectID, "source_files", sourceCount, "entities", len(graph.Entities), "relationships", len(graph.Relationships))
	return nil
}

type parseJob struct {
	index int
	file  repository.DiscoveredFile
}
type parseOutcome struct {
	index  int
	result parserpkg.ParseResult
	err    error
}

func (s *Service) parseFiles(ctx context.Context, runID string, discovered []repository.DiscoveredFile, total int) ([]parserpkg.ParseResult, error) {
	jobs := make(chan parseJob)
	outcomes := make(chan parseOutcome)
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				cacheKey := job.file.Record.Language + "\x00" + job.file.Record.Path + "\x00" + job.file.Record.ContentHash
				if cached, ok := s.cache.Load(cacheKey); ok {
					outcomes <- parseOutcome{index: job.index, result: cached.(parserpkg.ParseResult)}
					continue
				}
				content, err := os.ReadFile(job.file.AbsolutePath)
				if err != nil {
					outcomes <- parseOutcome{index: job.index, err: fmt.Errorf("read %s source file: %w", job.file.Record.Language, err)}
					continue
				}
				result, err := parserpkg.Parse(job.file.Record.Path, job.file.Record.Language, content)
				content = nil
				if err == nil {
					s.cache.Store(cacheKey, result)
				}
				outcomes <- parseOutcome{index: job.index, result: result, err: err}
			}
		}()
	}
	go func() {
		index := 0
	feed:
		for _, file := range discovered {
			if file.Record.Classification != "source" {
				continue
			}
			select {
			case <-ctx.Done():
				break feed
			case jobs <- parseJob{index: index, file: file}:
				index++
			}
		}
		close(jobs)
		wg.Wait()
		close(outcomes)
	}()
	results := make([]parserpkg.ParseResult, total)
	completed := 0
	entitiesScanned := 0
	edgesScanned := 0
	lastUpdate := time.Now()
	for outcome := range outcomes {
		if outcome.err != nil {
			return nil, outcome.err
		}
		results[outcome.index] = outcome.result
		entitiesScanned += len(outcome.result.Entities)
		edgesScanned += len(outcome.result.Imports) + len(outcome.result.References)
		completed++
		if completed == total || time.Since(lastUpdate) > 500*time.Millisecond {
			_ = s.store.UpdateRun(ctx, runID, "parsing", completed, total, entitiesScanned, edgesScanned,
				fmt.Sprintf("Scanned %d of %d source files", completed, total))
			lastUpdate = time.Now()
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// seedCounts totals the raw entity and edge seeds found across all parsed files
// (pre-resolution) — the "scanned" figures shown during analysis.
func seedCounts(parsed []parserpkg.ParseResult) (entities, edges int) {
	for _, result := range parsed {
		entities += len(result.Entities)
		edges += len(result.Imports) + len(result.References)
	}
	return entities, edges
}

func sanitizeError(err error, root string) string {
	message := err.Error()
	if root != "" {
		message = strings.ReplaceAll(message, root, "<repository>")
	}
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.ReplaceAll(message, "\n", " ")
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}
