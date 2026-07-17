package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/codeatlas/codeatlas/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	config.MaxConns = 12
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply database schema: %w", err)
	}
	return nil
}

func (s *Store) CreateProject(ctx context.Context, project model.Project, run model.AnalysisRun) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO projects (id, name, source_type, root_path, remote_url, git_ref, managed_clone, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		project.ID, project.Name, project.SourceType, project.RootPath, project.RemoteURL,
		project.GitRef, project.ManagedClone, model.ProjectQueued)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO analysis_runs (id, project_id, status, stage, message)
		VALUES ($1,$2,$3,'queued','Waiting for the local analyzer')`, run.ID, project.ID, model.RunQueued)
	if err != nil {
		return fmt.Errorf("insert analysis run: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) QueueAnalysis(ctx context.Context, projectID, runID string) error {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO analysis_runs (id, project_id, status, stage, message)
		SELECT $2, id, 'queued', 'queued', 'Waiting for the local analyzer'
		FROM projects WHERE id=$1`, projectID, runID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `UPDATE projects SET status='queued', updated_at=now() WHERE id=$1`, projectID)
	return err
}

func scanRun(row pgx.Row) (*model.AnalysisRun, error) {
	var run model.AnalysisRun
	err := row.Scan(&run.ID, &run.ProjectID, &run.Status, &run.Stage, &run.Completed,
		&run.Total, &run.Message, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &run, err
}

const runColumns = `id, project_id, status, stage, completed, total, message, error_message, started_at, completed_at, created_at`

func (s *Store) LatestRun(ctx context.Context, projectID string) (*model.AnalysisRun, error) {
	return scanRun(s.pool.QueryRow(ctx, `SELECT `+runColumns+` FROM analysis_runs WHERE project_id=$1 ORDER BY created_at DESC LIMIT 1`, projectID))
}

func (s *Store) ClaimRun(ctx context.Context) (*model.AnalysisRun, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		SELECT `+runColumns+` FROM analysis_runs
		WHERE status='queued' OR (status='running' AND lease_until < now())
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED LIMIT 1`)
	run, err := scanRun(row)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE analysis_runs SET status='running', stage='acquiring', started_at=COALESCE(started_at,now()),
			lease_until=now()+interval '45 seconds', message='Preparing repository'
		WHERE id=$1`, run.ID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE projects SET status='analyzing', updated_at=now() WHERE id=$1`, run.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	run.Status = model.RunRunning
	run.Stage = "acquiring"
	return run, nil
}

func (s *Store) UpdateRun(ctx context.Context, runID, stage string, completed, total int, message string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE analysis_runs SET stage=$2, completed=$3, total=$4, message=$5,
			lease_until=now()+interval '45 seconds' WHERE id=$1`, runID, stage, completed, total, message)
	return err
}

func (s *Store) FailRun(ctx context.Context, runID, projectID, safeMessage string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		UPDATE analysis_runs SET status='failed', stage='failed', error_message=$2,
			message='Analysis failed', lease_until=NULL, completed_at=now() WHERE id=$1`, runID, safeMessage)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE projects SET status='failed', updated_at=now() WHERE id=$1`, projectID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Project(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	err := s.pool.QueryRow(ctx, `
		SELECT id,name,source_type,root_path,remote_url,git_ref,current_commit,managed_clone,status,
			COALESCE(active_run_id,''),created_at,updated_at FROM projects WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.SourceType, &p.RootPath, &p.RemoteURL, &p.GitRef,
			&p.CurrentCommit, &p.ManagedClone, &p.Status, &p.ActiveRunID, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	run, err := s.LatestRun(ctx, id)
	if err == nil {
		p.LatestRun = run
	}
	return &p, nil
}

func (s *Store) Projects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id,p.name,p.source_type,p.root_path,p.remote_url,p.git_ref,p.current_commit,p.managed_clone,
			p.status,COALESCE(p.active_run_id,''),p.created_at,p.updated_at,
			r.id,r.status,r.stage,r.completed,r.total,r.message,r.error_message,r.started_at,r.completed_at,r.created_at
		FROM projects p
		LEFT JOIN LATERAL (
			SELECT * FROM analysis_runs ar WHERE ar.project_id=p.id ORDER BY ar.created_at DESC LIMIT 1
		) r ON true ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := make([]model.Project, 0)
	for rows.Next() {
		var p model.Project
		var runID, runStatus, runStage, runMessage, runError *string
		var completed, total *int
		var started, runCompleted, runCreated *time.Time
		if err := rows.Scan(&p.ID, &p.Name, &p.SourceType, &p.RootPath, &p.RemoteURL, &p.GitRef, &p.CurrentCommit,
			&p.ManagedClone, &p.Status, &p.ActiveRunID, &p.CreatedAt, &p.UpdatedAt,
			&runID, &runStatus, &runStage, &completed, &total, &runMessage, &runError, &started, &runCompleted, &runCreated); err != nil {
			return nil, err
		}
		if runID != nil {
			p.LatestRun = &model.AnalysisRun{ID: *runID, ProjectID: p.ID, Status: value(runStatus), Stage: value(runStage),
				Completed: intValue(completed), Total: intValue(total), Message: value(runMessage), ErrorMessage: value(runError),
				StartedAt: started, CompletedAt: runCompleted, CreatedAt: *runCreated}
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func intValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func (s *Store) UpdateProjectCommit(ctx context.Context, projectID, commit string) error {
	_, err := s.pool.Exec(ctx, `UPDATE projects SET current_commit=$2, updated_at=now() WHERE id=$1`, projectID, commit)
	return err
}

func (s *Store) ReplaceRunData(ctx context.Context, runID string, files []model.FileRecord, entities []model.Entity, relationships []model.Relationship) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM relationships WHERE run_id=$1`, runID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM entities WHERE run_id=$1`, runID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM files WHERE run_id=$1`, runID); err != nil {
		return err
	}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"files"}, []string{"id", "project_id", "run_id", "path", "language", "classification", "ignore_reason", "is_directory", "is_test", "size_bytes", "content_hash"},
		pgx.CopyFromSlice(len(files), func(i int) ([]any, error) {
			f := files[i]
			return []any{f.ID, f.ProjectID, f.RunID, f.Path, f.Language, f.Classification, f.IgnoreReason, f.IsDirectory, f.IsTest, f.SizeBytes, f.ContentHash}, nil
		}))
	if err != nil {
		return fmt.Errorf("insert file metadata: %w", err)
	}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"entities"}, []string{"id", "project_id", "run_id", "file_id", "kind", "name", "qualified_name", "language", "start_line", "start_column", "end_line", "end_column", "is_test", "metadata"},
		pgx.CopyFromSlice(len(entities), func(i int) ([]any, error) {
			e := entities[i]
			metadata, _ := json.Marshal(e.Metadata)
			return []any{e.ID, e.ProjectID, e.RunID, nullable(e.FileID), e.Kind, e.Name, e.QualifiedName, e.Language, e.Range.StartLine, e.Range.StartColumn, e.Range.EndLine, e.Range.EndColumn, e.IsTest, metadata}, nil
		}))
	if err != nil {
		return fmt.Errorf("insert entity metadata: %w", err)
	}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"relationships"}, []string{"id", "project_id", "run_id", "source_entity_id", "target_entity_id", "relationship_type", "confidence", "evidence_file_id", "start_line", "start_column", "end_line", "end_column", "resolution_state", "metadata"},
		pgx.CopyFromSlice(len(relationships), func(i int) ([]any, error) {
			r := relationships[i]
			metadata, _ := json.Marshal(r.Metadata)
			return []any{r.ID, r.ProjectID, r.RunID, r.SourceID, r.TargetID, r.Kind, r.Confidence, nullable(r.EvidenceFileID), r.Range.StartLine, r.Range.StartColumn, r.Range.EndLine, r.Range.EndColumn, r.Resolution, metadata}, nil
		}))
	if err != nil {
		return fmt.Errorf("insert relationship metadata: %w", err)
	}
	return tx.Commit(ctx)
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *Store) PromoteRun(ctx context.Context, runID, projectID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE analysis_runs SET status='ready',stage='ready',message='Analysis complete',lease_until=NULL,completed_at=now() WHERE id=$1`, runID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE projects SET active_run_id=$1,status='ready',updated_at=now() WHERE id=$2`, runID, projectID)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_, _ = s.pool.Exec(ctx, `
		DELETE FROM analysis_runs WHERE project_id=$1 AND id<>$2 AND status='ready' AND created_at < now()-interval '1 minute'`, projectID, runID)
	return nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Files(ctx context.Context, projectID, classification string, limit int) ([]model.FileRecord, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	rows, err := s.pool.Query(ctx, `
		SELECT f.id,f.project_id,f.run_id,f.path,f.language,f.classification,f.ignore_reason,f.is_directory,f.size_bytes,f.is_test
		FROM files f JOIN projects p ON p.active_run_id=f.run_id
		WHERE f.project_id=$1 AND ($2='' OR f.classification=$2)
		ORDER BY f.path LIMIT $3`, projectID, classification, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.FileRecord, 0)
	for rows.Next() {
		var f model.FileRecord
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.RunID, &f.Path, &f.Language, &f.Classification, &f.IgnoreReason,
			&f.IsDirectory, &f.SizeBytes, &f.IsTest); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

func scanEntity(row pgx.Row) (*model.Entity, error) {
	var e model.Entity
	var metadata []byte
	err := row.Scan(&e.ID, &e.ProjectID, &e.RunID, &e.FileID, &e.Kind, &e.Name, &e.QualifiedName, &e.Language,
		&e.Range.StartLine, &e.Range.StartColumn, &e.Range.EndLine, &e.Range.EndColumn, &e.IsTest, &metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(metadata, &e.Metadata)
	return &e, nil
}

const entityColumns = `e.id,e.project_id,e.run_id,COALESCE(e.file_id,''),e.kind,e.name,e.qualified_name,e.language,
	e.start_line,e.start_column,e.end_line,e.end_column,e.is_test,e.metadata`

func (s *Store) Entity(ctx context.Context, projectID, entityID string) (*model.Entity, error) {
	return scanEntity(s.pool.QueryRow(ctx, `SELECT `+entityColumns+` FROM entities e JOIN projects p ON p.active_run_id=e.run_id WHERE e.project_id=$1 AND e.id=$2`, projectID, entityID))
}

func (s *Store) Search(ctx context.Context, projectID, query string, limit int) ([]model.Entity, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.pool.Query(ctx, `SELECT `+entityColumns+` FROM entities e JOIN projects p ON p.active_run_id=e.run_id
		WHERE e.project_id=$1 AND (e.qualified_name ILIKE '%'||$2||'%' OR e.name ILIKE '%'||$2||'%')
		ORDER BY CASE WHEN lower(e.name)=lower($2) THEN 0 ELSE 1 END, similarity(e.qualified_name,$2) DESC, e.qualified_name LIMIT $3`, projectID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Entity, 0)
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *e)
	}
	return result, rows.Err()
}

type EvidenceRecord struct {
	Entity       model.Entity
	RootPath     string
	RelativePath string
	ContentHash  string
}

func (s *Store) Evidence(ctx context.Context, projectID, entityID string) (*EvidenceRecord, error) {
	var record EvidenceRecord
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		SELECT `+entityColumns+`,p.root_path,f.path,f.content_hash
		FROM entities e JOIN projects p ON p.active_run_id=e.run_id JOIN files f ON f.id=e.file_id
		WHERE e.project_id=$1 AND e.id=$2`, projectID, entityID).
		Scan(&record.Entity.ID, &record.Entity.ProjectID, &record.Entity.RunID, &record.Entity.FileID, &record.Entity.Kind,
			&record.Entity.Name, &record.Entity.QualifiedName, &record.Entity.Language, &record.Entity.Range.StartLine,
			&record.Entity.Range.StartColumn, &record.Entity.Range.EndLine, &record.Entity.Range.EndColumn, &record.Entity.IsTest,
			&metadata, &record.RootPath, &record.RelativePath, &record.ContentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(metadata, &record.Entity.Metadata)
	return &record, nil
}
