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

// seedImpactGraph builds a two-module codebase with cross-file edges so impact
// traversal has real dependents/dependencies to find:
//
//	app/handler.go:  rtUsers --route--> handleUsers --calls--> listUsers
//	services/user.go: listUsers --calls--> validate
//	app/handler.go   --imports--> services/user.go
func seedImpactGraph(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	project := model.Project{ID: "p1", Name: "demo", SourceType: model.SourceLocal, RootPath: t.TempDir()}
	run := model.AnalysisRun{ID: "run1", ProjectID: "p1"}
	if err := s.CreateProject(ctx, project, run); err != nil {
		t.Fatalf("create project: %v", err)
	}

	files := []model.FileRecord{
		{ID: "fh", ProjectID: "p1", RunID: "run1", Path: "app/handler.go", Language: "go", Classification: "source"},
		{ID: "fs", ProjectID: "p1", RunID: "run1", Path: "services/user.go", Language: "go", Classification: "source"},
	}
	entity := func(id, kind, name, fileID string) model.Entity {
		return model.Entity{ID: id, ProjectID: "p1", RunID: "run1", FileID: fileID, Kind: kind, Name: name, QualifiedName: name, Language: "go"}
	}
	entities := []model.Entity{
		entity("mApp", "module", "app", ""),
		entity("mSvc", "module", "services", ""),
		entity("feHandler", "file", "handler.go", "fh"),
		entity("feSvc", "file", "user.go", "fs"),
		entity("handleUsers", "function", "handleUsers", "fh"),
		entity("rtUsers", "route", "GET /users", "fh"),
		entity("listUsers", "function", "listUsers", "fs"),
		entity("validate", "function", "validate", "fs"),
	}
	rel := func(id, kind, src, dst string) model.Relationship {
		return model.Relationship{ID: id, ProjectID: "p1", RunID: "run1", SourceID: src, TargetID: dst, Kind: kind, Confidence: 1, Resolution: "resolved"}
	}
	relationships := []model.Relationship{
		rel("c1", "contains", "mApp", "feHandler"),
		rel("c2", "contains", "mSvc", "feSvc"),
		rel("d1", "defines", "feHandler", "handleUsers"),
		rel("d2", "defines", "feHandler", "rtUsers"),
		rel("d3", "defines", "feSvc", "listUsers"),
		rel("d4", "defines", "feSvc", "validate"),
		rel("h1", "handles_route", "rtUsers", "handleUsers"),
		rel("k1", "calls", "handleUsers", "listUsers"),
		rel("k2", "calls", "listUsers", "validate"),
		rel("i1", "imports", "feHandler", "feSvc"),
	}
	if err := s.ReplaceRunData(ctx, "run1", files, entities, relationships); err != nil {
		t.Fatalf("replace run data: %v", err)
	}
	if err := s.PromoteRun(ctx, "run1", "p1"); err != nil {
		t.Fatalf("promote run: %v", err)
	}
}

func TestImpactMapFromSymbol(t *testing.T) {
	s := newTestStore(t)
	seedImpactGraph(t, s)

	graph, err := s.ImpactMap(context.Background(), "p1", "listUsers", 6, 200)
	if err != nil {
		t.Fatalf("impact map: %v", err)
	}
	if got := directionOf(graph, "listUsers"); got != "root" {
		t.Errorf("listUsers direction = %q, want root", got)
	}
	// Callers upstream break when listUsers changes.
	if got := directionOf(graph, "handleUsers"); got != "dependent" {
		t.Errorf("handleUsers direction = %q, want dependent", got)
	}
	if got := directionOf(graph, "rtUsers"); got != "dependent" {
		t.Errorf("rtUsers direction = %q, want dependent", got)
	}
	// What listUsers relies on downstream.
	if got := directionOf(graph, "validate"); got != "dependency" {
		t.Errorf("validate direction = %q, want dependency", got)
	}
}

func TestImpactMapFromFile(t *testing.T) {
	s := newTestStore(t)
	seedImpactGraph(t, s)

	// Changing services/user.go should flag the app side that imports/calls it.
	graph, err := s.ImpactMap(context.Background(), "p1", "feSvc", 6, 200)
	if err != nil {
		t.Fatalf("impact map: %v", err)
	}
	if got := directionOf(graph, "listUsers"); got != "root" {
		t.Errorf("listUsers (a defined symbol of the file) direction = %q, want root", got)
	}
	if got := directionOf(graph, "handleUsers"); got != "dependent" {
		t.Errorf("handleUsers direction = %q, want dependent", got)
	}
	if got := directionOf(graph, "feHandler"); got != "dependent" {
		t.Errorf("feHandler (imports the file) direction = %q, want dependent", got)
	}
}

func TestImpactMapFromModule(t *testing.T) {
	s := newTestStore(t)
	seedImpactGraph(t, s)

	graph, err := s.ImpactMap(context.Background(), "p1", "mSvc", 6, 200)
	if err != nil {
		t.Fatalf("impact map: %v", err)
	}
	// The services module's dependents include the app file/symbols that use it.
	if !hasNode(graph, "handleUsers") || directionOf(graph, "handleUsers") != "dependent" {
		t.Errorf("expected handleUsers as a dependent of module services, got %q", directionOf(graph, "handleUsers"))
	}
	if !hasNode(graph, "feHandler") || directionOf(graph, "feHandler") != "dependent" {
		t.Errorf("expected feHandler as a dependent of module services, got %q", directionOf(graph, "feHandler"))
	}
}

func TestEntitiesInRanges(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.CreateProject(ctx,
		model.Project{ID: "p1", Name: "demo", SourceType: model.SourceLocal, RootPath: t.TempDir()},
		model.AnalysisRun{ID: "run1", ProjectID: "p1"}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	files := []model.FileRecord{
		{ID: "f1", ProjectID: "p1", RunID: "run1", Path: "src/orders.ts", Language: "typescript", Classification: "source"},
	}
	fn := func(id, name string, start, end int) model.Entity {
		return model.Entity{ID: id, ProjectID: "p1", RunID: "run1", FileID: "f1", Kind: "function",
			Name: name, QualifiedName: "src/orders.ts::" + name, Language: "typescript",
			Range: model.Range{StartLine: start, EndLine: end}}
	}
	entities := []model.Entity{
		fn("fetchOrders", "fetchOrders", 10, 20),
		fn("helper", "helper", 30, 40),
		// The file entity has no real range and must never match a positive range.
		{ID: "fe", ProjectID: "p1", RunID: "run1", FileID: "f1", Kind: "file", Name: "orders.ts", QualifiedName: "src/orders.ts"},
	}
	if err := s.ReplaceRunData(ctx, "run1", files, entities, nil); err != nil {
		t.Fatalf("replace run data: %v", err)
	}
	if err := s.PromoteRun(ctx, "run1", "p1"); err != nil {
		t.Fatalf("promote run: %v", err)
	}

	// A change at lines 12–14 overlaps only fetchOrders.
	ids, err := s.EntitiesInRanges(ctx, "p1", "src/orders.ts", [][2]int{{12, 14}})
	if err != nil {
		t.Fatalf("entities in ranges: %v", err)
	}
	if len(ids) != 1 || ids[0] != "fetchOrders" {
		t.Errorf("got %v, want [fetchOrders]", ids)
	}

	// A change spanning 18–32 overlaps both functions (never the file entity).
	ids, err = s.EntitiesInRanges(ctx, "p1", "src/orders.ts", [][2]int{{18, 32}})
	if err != nil {
		t.Fatalf("entities in ranges: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("got %v, want both functions", ids)
	}

	// A change above any symbol (imports) matches nothing → callers fall back to the file entity.
	ids, err = s.EntitiesInRanges(ctx, "p1", "src/orders.ts", [][2]int{{1, 3}})
	if err != nil {
		t.Fatalf("entities in ranges: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("got %v, want none", ids)
	}
	fileID, err := s.FileEntityID(ctx, "p1", "src/orders.ts")
	if err != nil || fileID != "fe" {
		t.Errorf("file entity id = %q, err = %v, want fe", fileID, err)
	}
}

func TestChangeImpact(t *testing.T) {
	s := newTestStore(t)
	seedImpactGraph(t, s)

	// Pretend the diff touched listUsers. Its change ripple: callers break, callees are relied on.
	graph, err := s.ChangeImpact(context.Background(), "p1", []string{"listUsers"}, 6, 200)
	if err != nil {
		t.Fatalf("change impact: %v", err)
	}
	if got := directionOf(graph, "listUsers"); got != "changed" {
		t.Errorf("listUsers direction = %q, want changed", got)
	}
	if got := directionOf(graph, "handleUsers"); got != "dependent" {
		t.Errorf("handleUsers direction = %q, want dependent", got)
	}
	if got := directionOf(graph, "validate"); got != "dependency" {
		t.Errorf("validate direction = %q, want dependency", got)
	}
}

func directionOf(g model.Graph, id string) string {
	for _, n := range g.Nodes {
		if n.ID == id {
			return n.Direction
		}
	}
	return ""
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
