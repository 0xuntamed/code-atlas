package analyzer

import (
	"fmt"

	"github.com/codeatlas/codeatlas/internal/id"
	"github.com/codeatlas/codeatlas/internal/model"
)

func (b *graphBuilder) addImports() {
	for _, result := range b.parsed {
		sourceFileID := b.fileEntityIDs[result.Path]
		fileRecord := b.fileRecords[result.Path]
		for _, imported := range result.Imports {
			targetID, resolution, confidence := b.resolveImportedFile(result, imported.Specifier)
			b.graph.Relationships = append(
				b.graph.Relationships,
				newRelationship(
					b.projectID,
					b.runID,
					sourceFileID,
					targetID,
					"imports",
					confidence,
					fileRecord.ID,
					imported.Range,
					resolution,
					map[string]any{"specifier": imported.Specifier},
				),
			)
		}
	}
}

func (b *graphBuilder) addReferences() {
	for _, result := range b.parsed {
		fileRecord := b.fileRecords[result.Path]
		for _, reference := range result.References {
			sourceID := b.referenceSourceID(result.Path, reference.FromKey)
			targetID, resolution, confidence := resolveReference(
				reference.Target,
				result.Path,
				b.importsByFile[result.Path],
				b.byName,
				b.byQualified,
			)
			if targetID == "" {
				targetID = b.unresolvedEntity(result.Path, reference.Target)
				resolution = "unresolved"
				confidence = 0.35
			}
			if reference.Confidence > 0 && reference.Confidence < confidence {
				confidence = reference.Confidence
			}

			b.graph.Relationships = append(
				b.graph.Relationships,
				newRelationship(
					b.projectID,
					b.runID,
					sourceID,
					targetID,
					reference.Kind,
					confidence,
					fileRecord.ID,
					reference.Range,
					resolution,
					reference.Metadata,
				),
			)
		}
	}
}

func (b *graphBuilder) referenceSourceID(filePath, fromKey string) string {
	sourceID := b.fileEntityIDs[filePath]
	if fromKey == "file" {
		return sourceID
	}
	if entityID := b.seedIDs[seedLookupKey(filePath, fromKey)]; entityID != "" {
		return entityID
	}
	return sourceID
}

func (b *graphBuilder) externalEntity(name string) string {
	key := "external:" + name
	if existing := b.syntheticIDs[key]; existing != "" {
		return existing
	}

	entityID := id.Stable(b.runID, key)
	b.syntheticIDs[key] = entityID
	b.graph.Entities = append(b.graph.Entities, model.Entity{
		ID:            entityID,
		ProjectID:     b.projectID,
		RunID:         b.runID,
		Kind:          "external_symbol",
		Name:          name,
		QualifiedName: name,
		Metadata:      map[string]any{"external": true},
	})
	return entityID
}

func (b *graphBuilder) unresolvedEntity(filePath, name string) string {
	key := "unresolved:" + filePath + ":" + name
	if existing := b.syntheticIDs[key]; existing != "" {
		return existing
	}

	entityID := id.Stable(b.runID, key)
	b.syntheticIDs[key] = entityID
	b.graph.Entities = append(b.graph.Entities, model.Entity{
		ID:            entityID,
		ProjectID:     b.projectID,
		RunID:         b.runID,
		Kind:          "unresolved_symbol",
		Name:          name,
		QualifiedName: fmt.Sprintf("%s::unresolved::%s", filePath, name),
		Metadata:      map[string]any{"unresolved": true},
	})
	return entityID
}

// moduleEdgeKinds are the symbol/file relationship kinds that roll up into
// module-to-module edges for the architecture overview.
var moduleEdgeKinds = map[string]bool{"imports": true, "calls": true, "depends_on": true}

// addModuleEdges precomputes aggregate module-to-module edges from the
// symbol/file relationships. Storing these (with a "module_" kind prefix so they
// never mix into symbol-level traversals) lets the architecture overview read a
// few hundred edges directly instead of joining every symbol to its module at
// query time — which was the overview's dominant cost on large repositories.
func (b *graphBuilder) addModuleEdges() {
	type edgeKey struct{ source, target, kind string }
	type edgeAgg struct {
		count       int
		confidence  float64
		allResolved bool
	}
	aggregates := make(map[edgeKey]*edgeAgg)
	for _, relationship := range b.graph.Relationships {
		if !moduleEdgeKinds[relationship.Kind] {
			continue
		}
		sourceModule := b.entityModule[relationship.SourceID]
		targetModule := b.entityModule[relationship.TargetID]
		if sourceModule == "" || targetModule == "" || sourceModule == targetModule {
			continue
		}
		key := edgeKey{sourceModule, targetModule, relationship.Kind}
		agg := aggregates[key]
		if agg == nil {
			agg = &edgeAgg{allResolved: true}
			aggregates[key] = agg
		}
		agg.count++
		agg.confidence += relationship.Confidence
		if relationship.Resolution != "resolved" {
			agg.allResolved = false
		}
	}
	for key, agg := range aggregates {
		resolution := "inferred"
		if agg.allResolved {
			resolution = "resolved"
		}
		b.graph.Relationships = append(b.graph.Relationships, model.Relationship{
			ID:         id.Stable(b.runID, "module_edge", key.source, key.target, key.kind),
			ProjectID:  b.projectID,
			RunID:      b.runID,
			SourceID:   key.source,
			TargetID:   key.target,
			Kind:       "module_" + key.kind,
			Confidence: agg.confidence / float64(agg.count),
			Resolution: resolution,
			Metadata:   map[string]any{"aggregate": true, "relationshipCount": agg.count},
		})
	}
}

func dedupeRelationships(input []model.Relationship) []model.Relationship {
	seen := make(map[string]struct{}, len(input))
	result := make([]model.Relationship, 0, len(input))
	for _, relationship := range input {
		if _, exists := seen[relationship.ID]; exists {
			continue
		}
		seen[relationship.ID] = struct{}{}
		result = append(result, relationship)
	}
	return result
}

func newRelationship(
	projectID string,
	runID string,
	sourceID string,
	targetID string,
	kind string,
	confidence float64,
	fileID string,
	sourceRange model.Range,
	resolution string,
	metadata map[string]any,
) model.Relationship {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return model.Relationship{
		ID: id.Stable(
			runID,
			"relationship",
			sourceID,
			targetID,
			kind,
			fmt.Sprint(sourceRange.StartLine),
		),
		ProjectID:      projectID,
		RunID:          runID,
		SourceID:       sourceID,
		TargetID:       targetID,
		Kind:           kind,
		Confidence:     confidence,
		EvidenceFileID: fileID,
		Range:          sourceRange,
		Resolution:     resolution,
		Metadata:       metadata,
	}
}
