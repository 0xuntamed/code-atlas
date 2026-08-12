package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/codeatlas/codeatlas/internal/model"
)

const relationshipColumns = `r.id,r.project_id,r.run_id,r.source_entity_id,r.target_entity_id,r.relationship_type,
	r.confidence,COALESCE(r.evidence_file_id,''),r.start_line,r.start_column,r.end_line,r.end_column,r.resolution_state,r.metadata`

func scanRelationship(row scanner) (model.Relationship, error) {
	var r model.Relationship
	var metadata []byte
	err := row.Scan(&r.ID, &r.ProjectID, &r.RunID, &r.SourceID, &r.TargetID, &r.Kind, &r.Confidence, &r.EvidenceFileID,
		&r.Range.StartLine, &r.Range.StartColumn, &r.Range.EndLine, &r.Range.EndColumn, &r.Resolution, &metadata)
	if err == nil {
		_ = json.Unmarshal(metadata, &r.Metadata)
	}
	return r, err
}

// ArchitectureGraph returns a deliberately small, progressive view of a project.
// The root view contains module aggregates. Supplying a scope ID drills into one
// entity's immediate neighborhood instead of flattening the whole codebase.
func (s *Store) ArchitectureGraph(ctx context.Context, projectID, scopeID string, limit int) (model.Graph, error) {
	if limit <= 0 || limit > 120 {
		limit = 80
	}
	if scopeID != "" {
		return s.NeighborhoodGraph(ctx, projectID, scopeID, limit)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT `+entityColumns+`,
			count(DISTINCT file_entity.id) AS file_count,
			count(DISTINCT file_entity.id) FILTER (WHERE file_entity.is_test) AS test_file_count,
			count(DISTINCT symbol.id) FILTER (
				WHERE symbol.kind NOT IN ('module', 'file')
			) AS symbol_count,
			count(DISTINCT symbol.id) FILTER (WHERE symbol.kind = 'route') AS route_count
		FROM entities e
		JOIN projects p ON p.active_run_id=e.run_id
		LEFT JOIN relationships contains_edge
			ON contains_edge.run_id=e.run_id
			AND contains_edge.source_entity_id=e.id
			AND contains_edge.relationship_type='contains'
		LEFT JOIN entities file_entity
			ON file_entity.id=contains_edge.target_entity_id
			AND file_entity.kind='file'
		LEFT JOIN entities symbol
			ON symbol.run_id=e.run_id
			AND symbol.file_id=file_entity.file_id
		WHERE e.project_id=? AND e.kind='module'
		GROUP BY e.id,p.active_run_id
		ORDER BY route_count DESC,symbol_count DESC,e.qualified_name
		LIMIT ?`, projectID, limit+1)
	if err != nil {
		return model.Graph{}, err
	}
	defer rows.Close()
	nodes := make([]model.Entity, 0, limit)
	truncated := false
	for rows.Next() {
		var entity model.Entity
		var metadata []byte
		var isTest int
		var fileCount, testFileCount, symbolCount, routeCount int
		if err := rows.Scan(
			&entity.ID,
			&entity.ProjectID,
			&entity.RunID,
			&entity.FileID,
			&entity.Kind,
			&entity.Name,
			&entity.QualifiedName,
			&entity.Language,
			&entity.Range.StartLine,
			&entity.Range.StartColumn,
			&entity.Range.EndLine,
			&entity.Range.EndColumn,
			&isTest,
			&metadata,
			&fileCount,
			&testFileCount,
			&symbolCount,
			&routeCount,
		); err != nil {
			return model.Graph{}, err
		}
		_ = json.Unmarshal(metadata, &entity.Metadata)
		if entity.Metadata == nil {
			entity.Metadata = make(map[string]any)
		}
		entity.Metadata["fileCount"] = fileCount
		entity.Metadata["testFileCount"] = testFileCount
		entity.Metadata["symbolCount"] = symbolCount
		entity.Metadata["routeCount"] = routeCount
		entity.Metadata["expandable"] = true
		entity.IsTest = fileCount > 0 && testFileCount == fileCount
		if len(nodes) < limit {
			nodes = append(nodes, entity)
		} else {
			truncated = true
		}
	}
	if err := rows.Err(); err != nil {
		return model.Graph{}, err
	}
	edges, err := s.moduleEdges(ctx, projectID, nodes)
	return model.Graph{Nodes: nodes, Edges: edges, Truncated: truncated, Limit: limit}, err
}

// NeighborhoodGraph returns one root and its direct structural or behavioral
// neighbors. It is the expansion primitive used by the architecture explorer.
func (s *Store) NeighborhoodGraph(ctx context.Context, projectID, rootID string, limit int) (model.Graph, error) {
	root, err := s.Entity(ctx, projectID, rootID)
	if err != nil {
		return model.Graph{}, err
	}

	direction := "both"
	kinds := []string{"calls", "handles_route", "uses_middleware", "imports", "depends_on"}
	switch root.Kind {
	case "module":
		direction = "downstream"
		kinds = []string{"contains"}
	case "file":
		kinds = []string{"defines", "imports", "exports", "depends_on"}
	}

	return s.traversalGraph(ctx, projectID, rootID, direction, 1, limit, kinds)
}

func (s *Store) moduleEdges(ctx context.Context, projectID string, nodes []model.Entity) ([]model.Relationship, error) {
	if len(nodes) == 0 {
		return []model.Relationship{}, nil
	}

	moduleIDs := make([]string, len(nodes))
	for index := range nodes {
		moduleIDs[index] = nodes[index].ID
	}
	kinds := []string{"imports", "calls", "depends_on"}

	query := fmt.Sprintf(`
		WITH module_files AS (
			SELECT contains_edge.source_entity_id AS module_id,file_entity.file_id
			FROM relationships contains_edge
			JOIN projects p ON p.active_run_id=contains_edge.run_id
			JOIN entities file_entity ON file_entity.id=contains_edge.target_entity_id
			WHERE contains_edge.project_id=?
				AND contains_edge.source_entity_id IN (%s)
				AND contains_edge.relationship_type='contains'
		)
		SELECT source_module.module_id,target_module.module_id,
			rel.relationship_type,count(*) AS relationship_count,
			avg(rel.confidence) AS confidence,
			min(CASE WHEN rel.resolution_state='resolved' THEN 1 ELSE 0 END) AS fully_resolved
		FROM relationships rel
		JOIN projects p ON p.active_run_id=rel.run_id
		JOIN entities source_entity ON source_entity.id=rel.source_entity_id
		JOIN entities target_entity ON target_entity.id=rel.target_entity_id
		JOIN module_files source_module ON source_module.file_id=source_entity.file_id
		JOIN module_files target_module ON target_module.file_id=target_entity.file_id
		WHERE rel.project_id=?
			AND rel.relationship_type IN (%s)
			AND source_module.module_id<>target_module.module_id
		GROUP BY source_module.module_id,target_module.module_id,rel.relationship_type
		ORDER BY relationship_count DESC,source_module.module_id,target_module.module_id`,
		placeholders(len(moduleIDs)), placeholders(len(kinds)))

	args := []any{projectID}
	args = append(args, stringArgs(moduleIDs)...)
	args = append(args, projectID)
	args = append(args, stringArgs(kinds)...)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]model.Relationship, 0)
	for rows.Next() {
		var edge model.Relationship
		var count int
		var fullyResolved int
		if err := rows.Scan(
			&edge.SourceID,
			&edge.TargetID,
			&edge.Kind,
			&count,
			&edge.Confidence,
			&fullyResolved,
		); err != nil {
			return nil, err
		}
		edge.ID = fmt.Sprintf("aggregate:%s:%s:%s", edge.SourceID, edge.TargetID, edge.Kind)
		edge.ProjectID = projectID
		edge.Resolution = "inferred"
		if fullyResolved == 1 {
			edge.Resolution = "resolved"
		}
		edge.Metadata = map[string]any{"aggregate": true, "relationshipCount": count}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

func (s *Store) FlowGraph(ctx context.Context, projectID, rootID string, depth, limit int) (model.Graph, error) {
	return s.traversalGraph(ctx, projectID, rootID, "downstream", depth, limit,
		[]string{"handles_route", "uses_middleware", "calls"})
}

func (s *Store) ImpactGraph(ctx context.Context, projectID, rootID, direction string, depth, limit int) (model.Graph, error) {
	if direction != "upstream" && direction != "downstream" && direction != "both" {
		direction = "both"
	}
	return s.traversalGraph(ctx, projectID, rootID, direction, depth, limit,
		[]string{"calls", "handles_route", "uses_middleware", "imports", "depends_on"})
}

func (s *Store) traversalGraph(ctx context.Context, projectID, rootID, direction string, depth, limit int, kinds []string) (model.Graph, error) {
	if depth <= 0 {
		depth = 5
	}
	if depth > 12 {
		depth = 12
	}
	if limit <= 0 || limit > 500 {
		limit = 500
	}

	// The next node reached along an edge, and the predicate that selects
	// candidate edges, depend on the traversal direction. Entity IDs are hex,
	// so a '/'-delimited path string is a safe cycle guard.
	var nextExpr, joinPredicate string
	switch direction {
	case "upstream":
		nextExpr = "r.source_entity_id"
		joinPredicate = "r.target_entity_id = w.node_id"
	case "downstream":
		nextExpr = "r.target_entity_id"
		joinPredicate = "r.source_entity_id = w.node_id"
	default:
		nextExpr = "CASE WHEN r.source_entity_id = w.node_id THEN r.target_entity_id ELSE r.source_entity_id END"
		joinPredicate = "(r.source_entity_id = w.node_id OR r.target_entity_id = w.node_id)"
	}

	query := fmt.Sprintf(`
		WITH RECURSIVE walk(node_id, depth, path) AS (
			SELECT ?, 0, '/' || ? || '/'
			UNION ALL
			SELECT %[1]s, w.depth + 1, w.path || %[1]s || '/'
			FROM walk w
			JOIN relationships r
				ON r.run_id = (SELECT active_run_id FROM projects WHERE id = ?)
			   AND r.relationship_type IN (%[2]s)
			   AND %[3]s
			WHERE w.depth < ?
			  AND instr(w.path, '/' || %[1]s || '/') = 0
		),
		selected AS (
			SELECT node_id, min(depth) AS distance
			FROM walk GROUP BY node_id ORDER BY min(depth), node_id LIMIT ?
		)
		SELECT %[4]s, selected.distance
		FROM selected
		JOIN entities e ON e.id = selected.node_id
		JOIN projects p ON p.active_run_id = e.run_id
		WHERE e.project_id = ?
		ORDER BY selected.distance, e.qualified_name`,
		nextExpr, placeholders(len(kinds)), joinPredicate, entityColumns)

	args := []any{rootID, rootID, projectID}
	args = append(args, stringArgs(kinds)...)
	args = append(args, depth, limit+1, projectID)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return model.Graph{}, fmt.Errorf("traverse graph: %w", err)
	}
	defer rows.Close()
	nodes := make([]model.Entity, 0, limit)
	truncated := false
	for rows.Next() {
		var e model.Entity
		var metadata []byte
		var isTest int
		err := rows.Scan(&e.ID, &e.ProjectID, &e.RunID, &e.FileID, &e.Kind, &e.Name, &e.QualifiedName, &e.Language,
			&e.Range.StartLine, &e.Range.StartColumn, &e.Range.EndLine, &e.Range.EndColumn, &isTest, &metadata, &e.Distance)
		if err != nil {
			return model.Graph{}, err
		}
		e.IsTest = isTest != 0
		_ = json.Unmarshal(metadata, &e.Metadata)
		if len(nodes) < limit {
			nodes = append(nodes, e)
		} else {
			truncated = true
		}
	}
	if err := rows.Err(); err != nil {
		return model.Graph{}, err
	}
	if len(nodes) == 0 {
		return model.Graph{}, ErrNotFound
	}
	edges, err := s.edgesForNodes(ctx, projectID, nodes, kinds)
	return model.Graph{Nodes: nodes, Edges: edges, RootID: rootID, Truncated: truncated, Limit: limit, MaxDepth: depth}, err
}

func (s *Store) edgesForNodes(ctx context.Context, projectID string, nodes []model.Entity, kinds []string) ([]model.Relationship, error) {
	if len(nodes) == 0 {
		return []model.Relationship{}, nil
	}
	ids := make([]string, len(nodes))
	for i := range nodes {
		ids[i] = nodes[i].ID
	}
	idList := placeholders(len(ids))
	query := fmt.Sprintf(`SELECT %s FROM relationships r JOIN projects p ON p.active_run_id=r.run_id
		WHERE r.project_id=? AND r.source_entity_id IN (%s) AND r.target_entity_id IN (%s)
		AND r.relationship_type IN (%s) ORDER BY r.relationship_type,r.id`,
		relationshipColumns, idList, idList, placeholders(len(kinds)))

	args := []any{projectID}
	args = append(args, stringArgs(ids)...)
	args = append(args, stringArgs(ids)...)
	args = append(args, stringArgs(kinds)...)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	edges := make([]model.Relationship, 0)
	for rows.Next() {
		r, err := scanRelationship(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, r)
	}
	return edges, rows.Err()
}
