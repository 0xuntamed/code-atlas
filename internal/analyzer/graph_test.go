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
