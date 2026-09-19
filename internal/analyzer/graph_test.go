package analyzer

import (
	"testing"

	"github.com/codeatlas/codeatlas/internal/id"
	"github.com/codeatlas/codeatlas/internal/model"
	parserpkg "github.com/codeatlas/codeatlas/internal/parser"
	"github.com/codeatlas/codeatlas/internal/repository"
)

func TestBuildGraphResolvesLocalCallsAndKeepsUnknowns(t *testing.T) {
	runID := "run"
	projectID := "project"
	files := []repository.DiscoveredFile{
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "src/a.ts"), ProjectID: projectID, RunID: runID, Path: "src/a.ts", Language: "typescript", Classification: "source"}},
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "src/b.ts"), ProjectID: projectID, RunID: runID, Path: "src/b.ts", Language: "typescript", Classification: "source"}},
	}
	parsed := []parserpkg.ParseResult{
		{Path: "src/a.ts", Language: "typescript", Entities: []parserpkg.EntitySeed{{Key: "a", Kind: "function", Name: "entry", QualifiedName: "src/a.ts::entry"}}, Imports: []parserpkg.ImportSeed{{FromKey: "file", Specifier: "./b"}}, References: []parserpkg.ReferenceSeed{{FromKey: "a", Target: "helper", Kind: "calls", Confidence: .9}, {FromKey: "a", Target: "dynamicTarget", Kind: "calls", Confidence: .5}}},
		{Path: "src/b.ts", Language: "typescript", Entities: []parserpkg.EntitySeed{{Key: "b", Kind: "function", Name: "helper", QualifiedName: "src/b.ts::helper"}}},
	}
	graph := buildGraph(projectID, runID, files, parsed)
	resolved, unresolved := false, false
	for _, relationship := range graph.Relationships {
		if relationship.Kind == "calls" && relationship.Resolution == "resolved" {
			resolved = true
		}
		if relationship.Kind == "calls" && relationship.Resolution == "unresolved" {
			unresolved = true
		}
	}
	if !resolved {
		t.Fatal("expected local imported call to resolve")
	}
	if !unresolved {
		t.Fatal("expected unknown call to remain explicit")
	}
}

func TestBuildGraphPrecomputesModuleCounts(t *testing.T) {
	runID := "run"
	projectID := "project"
	files := []repository.DiscoveredFile{
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "src/app.ts"), ProjectID: projectID, RunID: runID, Path: "src/app.ts", Language: "typescript", Classification: "source"}},
	}
	parsed := []parserpkg.ParseResult{
		{Path: "src/app.ts", Language: "typescript", Entities: []parserpkg.EntitySeed{
			{Key: "h", Kind: "function", Name: "handler", QualifiedName: "src/app.ts::handler"},
			{Key: "u", Kind: "function", Name: "helper", QualifiedName: "src/app.ts::helper"},
			{Key: "r", Kind: "route", Name: "GET /users", QualifiedName: "src/app.ts::GET /users"},
		}},
	}
	graph := buildGraph(projectID, runID, files, parsed)
	var module *model.Entity
	for index := range graph.Entities {
		if graph.Entities[index].Kind == "module" {
			module = &graph.Entities[index]
		}
	}
	if module == nil {
		t.Fatal("no module entity produced")
	}
	if module.Metadata["fileCount"] != 1 {
		t.Errorf("fileCount = %v, want 1", module.Metadata["fileCount"])
	}
	if module.Metadata["symbolCount"] != 3 {
		t.Errorf("symbolCount = %v, want 3", module.Metadata["symbolCount"])
	}
	if module.Metadata["routeCount"] != 1 {
		t.Errorf("routeCount = %v, want 1", module.Metadata["routeCount"])
	}
}

func TestBuildGraphPrecomputesModuleEdges(t *testing.T) {
	runID := "run"
	projectID := "project"
	files := []repository.DiscoveredFile{
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "src/a.ts"), ProjectID: projectID, RunID: runID, Path: "src/a.ts", Language: "typescript", Classification: "source"}},
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "lib/b.ts"), ProjectID: projectID, RunID: runID, Path: "lib/b.ts", Language: "typescript", Classification: "source"}},
	}
	parsed := []parserpkg.ParseResult{
		{Path: "src/a.ts", Language: "typescript",
			Entities: []parserpkg.EntitySeed{{Key: "a", Kind: "function", Name: "entry", QualifiedName: "src/a.ts::entry"}},
			Imports:  []parserpkg.ImportSeed{{FromKey: "file", Specifier: "../lib/b"}},
			References: []parserpkg.ReferenceSeed{{FromKey: "a", Target: "helper", Kind: "calls", Confidence: .9}}},
		{Path: "lib/b.ts", Language: "typescript", Entities: []parserpkg.EntitySeed{{Key: "b", Kind: "function", Name: "helper", QualifiedName: "lib/b.ts::helper"}}},
	}
	graph := buildGraph(projectID, runID, files, parsed)
	var moduleCalls *model.Relationship
	for index := range graph.Relationships {
		rel := &graph.Relationships[index]
		if rel.Kind == "module_calls" {
			moduleCalls = rel
		}
		// module_* edges must connect two module entities, never symbols.
		if len(rel.Kind) > 7 && rel.Kind[:7] == "module_" {
			if rel.SourceID == rel.TargetID {
				t.Fatalf("module edge is a self-loop: %+v", rel)
			}
		}
	}
	if moduleCalls == nil {
		t.Fatal("expected a precomputed module_calls edge between src and lib")
	}
	if got := moduleCalls.Metadata["relationshipCount"]; got != 1 {
		t.Errorf("relationshipCount = %v, want 1", got)
	}
}

func TestDedupeEntitiesKeepsFirstPerID(t *testing.T) {
	input := []model.Entity{
		{ID: "a", Name: "first"},
		{ID: "b", Name: "other"},
		{ID: "a", Name: "duplicate"},
	}
	out := dedupeEntities(input)
	if len(out) != 2 {
		t.Fatalf("expected 2 unique entities, got %d", len(out))
	}
	if out[0].ID != "a" || out[0].Name != "first" {
		t.Errorf("expected the first 'a' to survive, got %+v", out[0])
	}
}

func TestBuildGraphMarksTestOnlyModules(t *testing.T) {
	runID := "run"
	projectID := "project"
	files := []repository.DiscoveredFile{
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "src/main.ts"), ProjectID: projectID, RunID: runID, Path: "src/main.ts", Language: "typescript", Classification: "source"}},
		{Record: model.FileRecord{ID: id.Stable(runID, "file", "tests/main.test.ts"), ProjectID: projectID, RunID: runID, Path: "tests/main.test.ts", Language: "typescript", Classification: "source", IsTest: true}},
	}
	graph := buildGraph(projectID, runID, files, nil)
	modules := make(map[string]model.Entity)
	for _, entity := range graph.Entities {
		if entity.Kind == "module" {
			modules[entity.QualifiedName] = entity
		}
	}
	if modules["src"].IsTest {
		t.Fatal("production module was marked as test-only")
	}
	if !modules["tests"].IsTest {
		t.Fatal("test-only module was not marked as test")
	}
}
