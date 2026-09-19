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

	// Per-module rollup counts, accumulated during the build and written into
	// each module's metadata by finalizeModuleStats. Precomputing these makes
	// the architecture overview a cheap read instead of a query-time
	// aggregation over every symbol in the repository.
	moduleFileCount   map[string]int
	moduleTestFiles   map[string]int
	moduleSymbolCount map[string]int
	moduleRouteCount  map[string]int
	// entityModule maps a file/symbol entity id to the module it belongs to, so
	// module-to-module edges can be aggregated in memory (see addModuleEdges)
	// instead of joining symbols to modules at query time.
	entityModule map[string]string
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
	builder.addModuleEdges()
	builder.finalizeModuleStats()
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
		parsedByPath:      make(map[string]parserpkg.ParseResult),
		syntheticIDs:      make(map[string]string),
		importsByFile:     make(map[string][]string),
		moduleFileCount:   make(map[string]int),
		moduleTestFiles:   make(map[string]int),
		moduleSymbolCount: make(map[string]int),
		moduleRouteCount:  make(map[string]int),
		entityModule:      make(map[string]string),
	}
}

func (b *graphBuilder) finish() builtGraph {
	b.graph.Entities = dedupeEntities(b.graph.Entities)
	sort.Slice(b.graph.Entities, func(left, right int) bool {
		return b.graph.Entities[left].QualifiedName < b.graph.Entities[right].QualifiedName
	})
	b.graph.Relationships = dedupeRelationships(b.graph.Relationships)
	return b.graph
}

// dedupeEntities keeps the first entity for each id. Entity ids are stable
// hashes of (run, file, symbol key); a parser can occasionally emit two
// symbols in one file that hash to the same key (e.g. same-named declarations),
// which would otherwise violate the entities primary key and fail the whole
// run. Relationships reference ids, so dropping an exact-id duplicate leaves no
// dangling edges — they simply point at the surviving entity.
func dedupeEntities(input []model.Entity) []model.Entity {
	seen := make(map[string]struct{}, len(input))
	result := make([]model.Entity, 0, len(input))
	for _, entity := range input {
		if _, exists := seen[entity.ID]; exists {
			continue
		}
		seen[entity.ID] = struct{}{}
		result = append(result, entity)
	}
	return result
}
