package analyzer

import (
	"sort"

	"github.com/codeatlas/codeatlas/internal/model"
	parserpkg "github.com/codeatlas/codeatlas/internal/parser"
	"github.com/codeatlas/codeatlas/internal/repository"
)

type builtGraph struct {
	Files         []model.FileRecord
	Entities      []model.Entity
	Relationships []model.Relationship
}

type graphBuilder struct {
	projectID string
	runID     string

	discovered []repository.DiscoveredFile
	parsed     []parserpkg.ParseResult
	graph      builtGraph

	fileRecords   map[string]model.FileRecord
	fileEntityIDs map[string]string
	seedIDs       map[string]string
	moduleIDs     map[string]string
	byName        map[string][]model.Entity
	byQualified   map[string]model.Entity
	parsedByPath  map[string]parserpkg.ParseResult
	syntheticIDs  map[string]string
	importsByFile map[string][]string
}

func buildGraph(
	projectID string,
	runID string,
	discovered []repository.DiscoveredFile,
	parsed []parserpkg.ParseResult,
) builtGraph {
	builder := newGraphBuilder(projectID, runID, discovered, parsed)
	builder.indexFiles()
	builder.addStructuralEntities()
	builder.addDeclarations()
	builder.addImports()
	builder.addReferences()
	return builder.finish()
}

func newGraphBuilder(
	projectID string,
	runID string,
	discovered []repository.DiscoveredFile,
	parsed []parserpkg.ParseResult,
) *graphBuilder {
	return &graphBuilder{
		projectID:  projectID,
		runID:      runID,
		discovered: discovered,
		parsed:     parsed,
		graph: builtGraph{
			Files:         make([]model.FileRecord, 0, len(discovered)),
			Entities:      make([]model.Entity, 0),
			Relationships: make([]model.Relationship, 0),
		},
		fileRecords:   make(map[string]model.FileRecord),
		fileEntityIDs: make(map[string]string),
		seedIDs:       make(map[string]string),
		moduleIDs:     make(map[string]string),
		byName:        make(map[string][]model.Entity),
		byQualified:   make(map[string]model.Entity),
		parsedByPath:  make(map[string]parserpkg.ParseResult),
		syntheticIDs:  make(map[string]string),
		importsByFile: make(map[string][]string),
	}
}

func (b *graphBuilder) finish() builtGraph {
	sort.Slice(b.graph.Entities, func(left, right int) bool {
		return b.graph.Entities[left].QualifiedName < b.graph.Entities[right].QualifiedName
	})
	b.graph.Relationships = dedupeRelationships(b.graph.Relationships)
	return b.graph
}
