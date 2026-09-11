package api

import (
	"net/http"

	"github.com/codeatlas/codeatlas/internal/model"
	"github.com/codeatlas/codeatlas/internal/repository"
)

type changesSummary struct {
	FilesChanged   int `json:"filesChanged"`
	SymbolsChanged int `json:"symbolsChanged"`
	RoutesAffected int `json:"routesAffected"`
	TestsAffected  int `json:"testsAffected"`
}

type changesResponse struct {
	Graph   model.Graph    `json:"graph"`
	Summary changesSummary `json:"summary"`
}

// changes reports the blast radius of the repository's uncommitted changes:
// it diffs the working tree against HEAD, maps each changed line range to the
// entities it touches, and runs the change-impact traversal over that seed set.
func (s *Server) changes(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	project, err := s.store.Project(r.Context(), projectID)
	if handleStoreError(w, err) {
		return
	}
	if project.SourceType != model.SourceLocal {
		writeError(w, http.StatusBadRequest, "review_unsupported",
			"Change review works on local repositories only")
		return
	}

	fileChanges, err := repository.WorkingTreeDiff(r.Context(), project.RootPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "diff_failed", err.Error())
		return
	}

	seeds := make([]string, 0)
	seen := make(map[string]bool)
	filesChanged := 0
	for _, change := range fileChanges {
		ids, err := s.seedsForFile(r, projectID, change)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database_error", "Could not map changes")
			return
		}
		if len(ids) == 0 {
			continue // a changed file not represented in the current graph
		}
		filesChanged++
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				seeds = append(seeds, id)
			}
		}
	}

	if len(seeds) == 0 {
		writeJSON(w, http.StatusOK, changesResponse{
			Graph:   model.Graph{Nodes: []model.Entity{}, Edges: []model.Relationship{}},
			Summary: changesSummary{FilesChanged: filesChanged},
		})
		return
	}

	graph, err := s.store.ChangeImpact(r.Context(), projectID, seeds, parseDepth(r, 4), parseLimit(r, 200, 500))
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, changesResponse{Graph: graph, Summary: summarize(graph, filesChanged)})
}

// seedsForFile maps one changed file to the entities to seed the blast radius
// from: the symbols its changed ranges overlap, or — if the edit lands outside
// any symbol (imports, top-level, a whole new file) — the file entity itself.
func (s *Server) seedsForFile(r *http.Request, projectID string, change repository.FileChange) ([]string, error) {
	ranges := make([][2]int, 0, len(change.Ranges))
	for _, span := range change.Ranges {
		ranges = append(ranges, [2]int{span.Start, span.End})
	}

	ids, err := s.store.EntitiesInRanges(r.Context(), projectID, change.Path, ranges)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		return ids, nil
	}

	fileID, err := s.store.FileEntityID(r.Context(), projectID, change.Path)
	if err != nil {
		return nil, err
	}
	if fileID == "" {
		return nil, nil
	}
	return []string{fileID}, nil
}

// summarize derives the reviewer-facing headline numbers from the result graph:
// how many symbols changed, and how many routes/tests sit in their blast radius.
func summarize(graph model.Graph, filesChanged int) changesSummary {
	summary := changesSummary{FilesChanged: filesChanged}
	for _, node := range graph.Nodes {
		if node.Direction == "changed" {
			summary.SymbolsChanged++
		}
		affected := node.Direction == "dependent" || node.Direction == "both"
		if node.Kind == "route" && (affected || node.Direction == "changed") {
			summary.RoutesAffected++
		}
		if node.IsTest && affected {
			summary.TestsAffected++
		}
	}
	return summary
}
