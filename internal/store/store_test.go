package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/codeatlas/codeatlas/internal/model"
)

// newTestStore opens a fresh SQLite store backed by a temp file and applies the
// schema. The pure-Go driver needs no external database process.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(s.Close)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

// seedGraph creates one project with a promoted run describing a tiny codebase:
// a module containing one file that defines two functions and a route, with a
// call edge (fn1 -> fn2) and a route handler edge (rt -> fn1).
func seedGraph(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	project := model.Project{ID: "p1", Name: "demo", SourceType: model.SourceLocal, RootPath: t.TempDir()}
	run := model.AnalysisRun{ID: "run1", ProjectID: "p1"}
	if err := s.CreateProject(ctx, project, run); err != nil {
		t.Fatalf("create project: %v", err)
	}

	files := []model.FileRecord{{
		ID: "file1", ProjectID: "p1", RunID: "run1", Path: "main.go",
		Language: "go", Classification: "source", ContentHash: "abc123",
	}}
	entity := func(id, kind, name string) model.Entity {
		fileID := "file1"
		if kind == "module" {
			fileID = ""
		}
		return model.Entity{ID: id, ProjectID: "p1", RunID: "run1", FileID: fileID, Kind: kind, Name: name, QualifiedName: name, Language: "go"}
	}
	entities := []model.Entity{
		entity("m", "module", "app"),
		entity("fe", "file", "main.go"),
		entity("fn1", "function", "handler"),
		entity("fn2", "function", "helper"),
		entity("rt", "route", "GET /users"),
	}
	rel := func(id, kind, src, dst string) model.Relationship {
		return model.Relationship{ID: id, ProjectID: "p1", RunID: "run1", SourceID: src, TargetID: dst, Kind: kind, Confidence: 1, Resolution: "resolved"}
	}
	relationships := []model.Relationship{
		rel("r1", "contains", "m", "fe"),
		rel("r2", "defines", "fe", "fn1"),
		rel("r3", "defines", "fe", "fn2"),
		rel("r4", "defines", "fe", "rt"),
		rel("r5", "calls", "fn1", "fn2"),
		rel("r6", "handles_route", "rt", "fn1"),
	}
	if err := s.ReplaceRunData(ctx, "run1", files, entities, relationships); err != nil {
		t.Fatalf("replace run data: %v", err)
	}
	if err := s.PromoteRun(ctx, "run1", "p1"); err != nil {
		t.Fatalf("promote run: %v", err)
	}
}

func TestArchitectureGraphAggregatesModules(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)

	graph, err := s.ArchitectureGraph(context.Background(), "p1", "", 80)
	if err != nil {
		t.Fatalf("architecture graph: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 module node, got %d", len(graph.Nodes))
	}
	m := graph.Nodes[0]
	if m.Kind != "module" {
		t.Fatalf("expected module node, got %q", m.Kind)
	}
	if got := m.Metadata["fileCount"]; got != 1 {
		t.Errorf("fileCount = %v, want 1", got)
	}
	if got := m.Metadata["symbolCount"]; got != 3 {
		t.Errorf("symbolCount = %v, want 3", got)
	}
	if got := m.Metadata["routeCount"]; got != 1 {
		t.Errorf("routeCount = %v, want 1", got)
	}
}

func TestFlowGraphWalksDownstream(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)

	graph, err := s.FlowGraph(context.Background(), "p1", "fn1", 6, 120)
	if err != nil {
		t.Fatalf("flow graph: %v", err)
	}
	if !hasNode(graph, "fn1") || !hasNode(graph, "fn2") {
		t.Fatalf("expected fn1 and fn2 in downstream flow, got %v", nodeIDs(graph))
	}
	if graph.RootID != "fn1" {
		t.Errorf("root = %q, want fn1", graph.RootID)
	}
}

func TestImpactGraphWalksBothDirections(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)

	// fn2 is called by fn1; the upstream impact of changing fn2 must include fn1.
	graph, err := s.ImpactGraph(context.Background(), "p1", "fn2", "both", 4, 120)
	if err != nil {
		t.Fatalf("impact graph: %v", err)
	}
	if !hasNode(graph, "fn1") {
		t.Fatalf("expected fn1 in impact of fn2, got %v", nodeIDs(graph))
	}
}

func TestSearchFindsBySubstring(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)

	results, err := s.Search(context.Background(), "p1", "hand", 30)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected a match for 'hand'")
	}
	if results[0].Name != "handler" {
		t.Errorf("top result = %q, want handler", results[0].Name)
	}
}

func TestPromoteKeepsPreviousGraphOnQueuedRun(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)
	ctx := context.Background()

	// A newly queued run must not affect reads until it is promoted.
	if err := s.QueueAnalysis(ctx, "p1", "run2"); err != nil {
		t.Fatalf("queue analysis: %v", err)
	}
	graph, err := s.ArchitectureGraph(ctx, "p1", "", 80)
	if err != nil {
		t.Fatalf("architecture graph after queue: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected previous graph intact, got %d nodes", len(graph.Nodes))
	}
}

func TestProjectsAndEntityAndEvidence(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)
	ctx := context.Background()

	projects, err := s.Projects(ctx)
	if err != nil || len(projects) != 1 {
		t.Fatalf("projects = %v, err = %v", projects, err)
	}
	if projects[0].ActiveRunID != "run1" || projects[0].LatestRun == nil {
		t.Errorf("project not promoted/latest run missing: %+v", projects[0])
	}

	entity, err := s.Entity(ctx, "p1", "fn1")
	if err != nil || entity.Name != "handler" {
		t.Fatalf("entity = %+v, err = %v", entity, err)
	}

	evidence, err := s.Evidence(ctx, "p1", "fn1")
	if err != nil {
		t.Fatalf("evidence: %v", err)
	}
	if evidence.RelativePath != "main.go" || evidence.ContentHash != "abc123" {
		t.Errorf("evidence = %+v", evidence)
	}
}

func TestDeleteProjectCascades(t *testing.T) {
	s := newTestStore(t)
	seedGraph(t, s)
	ctx := context.Background()

	if err := s.DeleteProject(ctx, "p1"); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	projects, err := s.Projects(ctx)
	if err != nil {
		t.Fatalf("projects: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected no projects after delete, got %d", len(projects))
	}
	if err := s.DeleteProject(ctx, "p1"); err != ErrNotFound {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}
}

func hasNode(g model.Graph, id string) bool {
	for _, n := range g.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func nodeIDs(g model.Graph) []string {
	ids := make([]string, len(g.Nodes))
	for i, n := range g.Nodes {
		ids[i] = n.ID
	}
	return ids
}
