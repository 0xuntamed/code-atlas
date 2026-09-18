package analyzer

import (
	"path"

	"github.com/codeatlas/codeatlas/internal/id"
	"github.com/codeatlas/codeatlas/internal/model"
)

func (b *graphBuilder) indexFiles() {
	for _, item := range b.discovered {
		record := item.Record
		b.graph.Files = append(b.graph.Files, record)
		b.fileRecords[record.Path] = record
	}
	for _, result := range b.parsed {
		b.parsedByPath[result.Path] = result
	}
}

func (b *graphBuilder) addStructuralEntities() {
	moduleHasFiles := make(map[string]bool)
	moduleHasProductionFiles := make(map[string]bool)
	for _, item := range b.discovered {
		record := item.Record
		if record.Classification != "source" {
			continue
		}

		moduleID := b.ensureModule(record.Path)
		moduleHasFiles[moduleID] = true
		b.moduleFileCount[moduleID]++
		if record.IsTest {
			b.moduleTestFiles[moduleID]++
		} else {
			moduleHasProductionFiles[moduleID] = true
		}
		fileEntity := b.newFileEntity(record)
		b.fileEntityIDs[record.Path] = fileEntity.ID
		b.entityModule[fileEntity.ID] = moduleID
		b.graph.Entities = append(b.graph.Entities, fileEntity)
		indexEntity(fileEntity, b.byName, b.byQualified)
		b.graph.Relationships = append(
			b.graph.Relationships,
			newRelationship(
				b.projectID,
				b.runID,
				moduleID,
				fileEntity.ID,
				"contains",
				1,
				"",
				model.Range{},
				"resolved",
				nil,
			),
		)
	}

	for index := range b.graph.Entities {
		entity := &b.graph.Entities[index]
		if entity.Kind == "module" && moduleHasFiles[entity.ID] && !moduleHasProductionFiles[entity.ID] {
			entity.IsTest = true
		}
	}
}

func (b *graphBuilder) ensureModule(filePath string) string {
	directory := path.Dir(filePath)
	if directory == "." {
		directory = "(root)"
	}
	if existingID := b.moduleIDs[directory]; existingID != "" {
		return existingID
	}

	moduleID := id.Stable(b.runID, "module", directory)
	b.moduleIDs[directory] = moduleID
	b.graph.Entities = append(b.graph.Entities, model.Entity{
		ID:            moduleID,
		ProjectID:     b.projectID,
		RunID:         b.runID,
		Kind:          "module",
		Name:          path.Base(directory),
		QualifiedName: directory,
		Metadata:      map[string]any{"directory": directory},
	})
	return moduleID
}

func (b *graphBuilder) newFileEntity(record model.FileRecord) model.Entity {
	metadata := map[string]any{"path": record.Path}
	if parsedResult, ok := b.parsedByPath[record.Path]; ok && parsedResult.HasSyntaxErrors {
		metadata["hasSyntaxErrors"] = true
	}
	return model.Entity{
		ID:            id.Stable(b.runID, "entity:file", record.Path),
		ProjectID:     b.projectID,
		RunID:         b.runID,
		FileID:        record.ID,
		Kind:          "file",
		Name:          path.Base(record.Path),
		QualifiedName: record.Path,
		Language:      record.Language,
		IsTest:        record.IsTest,
		Metadata:      metadata,
	}
}

func (b *graphBuilder) addDeclarations() {
	for _, result := range b.parsed {
		fileRecord, exists := b.fileRecords[result.Path]
		if !exists {
			continue
		}

		moduleID := b.moduleIDForPath(result.Path)
		for _, seed := range result.Entities {
			entityID := id.Stable(b.runID, result.Path, seed.Key)
			b.seedIDs[seedLookupKey(result.Path, seed.Key)] = entityID
			b.entityModule[entityID] = moduleID
			b.moduleSymbolCount[moduleID]++
			if seed.Kind == "route" {
				b.moduleRouteCount[moduleID]++
			}
			entity := model.Entity{
				ID:            entityID,
				ProjectID:     b.projectID,
				RunID:         b.runID,
				FileID:        fileRecord.ID,
				Kind:          seed.Kind,
				Name:          seed.Name,
				QualifiedName: seed.QualifiedName,
				Language:      result.Language,
				Range:         seed.Range,
				Metadata:      seed.Metadata,
				IsTest:        fileRecord.IsTest,
			}
			b.graph.Entities = append(b.graph.Entities, entity)
			indexEntity(entity, b.byName, b.byQualified)
			b.graph.Relationships = append(
				b.graph.Relationships,
				newRelationship(
					b.projectID,
					b.runID,
					b.fileEntityIDs[result.Path],
					entityID,
					"defines",
					1,
					fileRecord.ID,
					seed.Range,
					"resolved",
					nil,
				),
			)
		}
	}
}

// moduleIDForPath returns the module id for a file's directory, matching the
// bucketing ensureModule uses ("(root)" for top-level files).
func (b *graphBuilder) moduleIDForPath(filePath string) string {
	directory := path.Dir(filePath)
	if directory == "." {
		directory = "(root)"
	}
	return b.moduleIDs[directory]
}

// finalizeModuleStats writes the accumulated per-module rollup counts into each
// module entity's metadata, so the architecture overview can read them directly
// instead of aggregating symbols at query time.
func (b *graphBuilder) finalizeModuleStats() {
	for index := range b.graph.Entities {
		entity := &b.graph.Entities[index]
		if entity.Kind != "module" {
			continue
		}
		if entity.Metadata == nil {
			entity.Metadata = map[string]any{}
		}
		entity.Metadata["fileCount"] = b.moduleFileCount[entity.ID]
		entity.Metadata["testFileCount"] = b.moduleTestFiles[entity.ID]
		entity.Metadata["symbolCount"] = b.moduleSymbolCount[entity.ID]
		entity.Metadata["routeCount"] = b.moduleRouteCount[entity.ID]
		entity.Metadata["expandable"] = true
	}
}

func seedLookupKey(filePath, seedKey string) string {
	return filePath + "\x00" + seedKey
}
