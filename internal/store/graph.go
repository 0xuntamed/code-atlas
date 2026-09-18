package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

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

	// The per-module rollup counts (files, symbols, routes) are precomputed at
	// analysis time and stored in each module's metadata (see
	// analyzer.finalizeModuleStats). So the overview is a cheap indexed read of
	// the module rows plus an in-memory sort — not a query-time aggregation over
	// every symbol in the repository, which was O(n^2) and took minutes on large
	// repos. There are at most a few thousand modules, so sorting in Go is fine.
	rows, err := s.db.QueryContext(ctx, `SELECT `+entityColumns+`
		FROM entities e JOIN projects p ON p.active_run_id=e.run_id
		WHERE e.project_id=? AND e.kind='module'`, projectID)
	if err != nil {
		return model.Graph{}, err
	}
	defer rows.Close()
	modules := make([]model.Entity, 0, 256)
	for rows.Next() {
		entity, err := scanEntity(rows)
		if err != nil {
			return model.Graph{}, err
		}
		if entity.Metadata == nil {
			entity.Metadata = make(map[string]any)
		}
		entity.Metadata["expandable"] = true
		modules = append(modules, *entity)
	}
	if err := rows.Err(); err != nil {
		return model.Graph{}, err
	}
	sort.SliceStable(modules, func(left, right int) bool {
		if rl, rr := metaInt(modules[left].Metadata, "routeCount"), metaInt(modules[right].Metadata, "routeCount"); rl != rr {
			return rl > rr
		}
		if sl, sr := metaInt(modules[left].Metadata, "symbolCount"), metaInt(modules[right].Metadata, "symbolCount"); sl != sr {
			return sl > sr
		}
		return modules[left].QualifiedName < modules[right].QualifiedName
	})
	truncated := len(modules) > limit
	if truncated {
		modules = modules[:limit]
	}
	edges, err := s.moduleEdges(ctx, projectID, modules)
	return model.Graph{Nodes: modules, Edges: edges, Truncated: truncated, Limit: limit}, err
}

// metaInt reads an integer from decoded JSON metadata, where numbers arrive as
// float64.
func metaInt(metadata map[string]any, key string) int {
	switch value := metadata[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	}
	return 0
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

	// Scope every scan to the project's active run explicitly. Filtering on
	// (project_id, run_id) lets the relationships indexes seek just that run's
	// rows instead of scanning every run's copy of the graph — the difference
	// between reading ~200k rows and every superseded run's rows combined.
	var activeRun string
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(active_run_id,'') FROM projects WHERE id=?`, projectID).Scan(&activeRun); err != nil {
		return nil, err
	}
	if activeRun == "" {
		return []model.Relationship{}, nil
	}

	moduleIDs := make([]string, len(nodes))
	for index := range nodes {
		moduleIDs[index] = nodes[index].ID
	}
	// The aggregate module-to-module edges are precomputed at analysis time and
	// stored with a "module_" kind prefix (see analyzer.addModuleEdges), so this
	// reads a few hundred edges by an indexed (project_id, run_id, source) seek
	// rather than aggregating every symbol relationship at query time.
	kinds := []string{"module_imports", "module_calls", "module_depends_on"}
	query := fmt.Sprintf(`
		SELECT source_entity_id,target_entity_id,relationship_type,confidence,resolution_state,metadata
		FROM relationships
		WHERE project_id=? AND run_id=?
			AND relationship_type IN (%s)
			AND source_entity_id IN (%s)
			AND target_entity_id IN (%s)
		ORDER BY relationship_type,source_entity_id,target_entity_id`,
		placeholders(len(kinds)), placeholders(len(moduleIDs)), placeholders(len(moduleIDs)))

	args := []any{projectID, activeRun}
	args = append(args, stringArgs(kinds)...)
	args = append(args, stringArgs(moduleIDs)...)
	args = append(args, stringArgs(moduleIDs)...)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]model.Relationship, 0)
	for rows.Next() {
		var edge model.Relationship
		var kind string
		var metadata []byte
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &kind, &edge.Confidence, &edge.Resolution, &metadata); err != nil {
			return nil, err
		}
		edge.Kind = strings.TrimPrefix(kind, "module_")
		edge.ID = fmt.Sprintf("aggregate:%s:%s:%s", edge.SourceID, edge.TargetID, edge.Kind)
		edge.ProjectID = projectID
		_ = json.Unmarshal(metadata, &edge.Metadata)
		if edge.Metadata == nil {
			edge.Metadata = map[string]any{}
		}
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

// impactEdgeKinds are the relationship kinds a change propagates along.
var impactEdgeKinds = []string{"calls", "handles_route", "uses_middleware", "imports", "depends_on"}

func clampDepth(depth int) int {
	if depth <= 0 {
		return 5
	}
	if depth > 12 {
		return 12
	}
	return depth
}

func clampNodes(limit int) int {
	if limit <= 0 || limit > 500 {
		return 500
	}
	return limit
}

func (s *Store) traversalGraph(ctx context.Context, projectID, rootID, direction string, depth, limit int, kinds []string) (model.Graph, error) {
	depth, limit = clampDepth(depth), clampNodes(limit)
	nodes, truncated, err := s.traversalNodes(ctx, projectID, []string{rootID}, direction, depth, limit, kinds)
	if err != nil {
		return model.Graph{}, err
	}
	if len(nodes) == 0 {
		return model.Graph{}, ErrNotFound
	}
	edges, err := s.edgesForNodes(ctx, projectID, nodes, kinds)
	return model.Graph{Nodes: nodes, Edges: edges, RootID: rootID, Truncated: truncated, Limit: limit, MaxDepth: depth}, err
}

// ImpactMap returns the blast radius of a change to rootID: everything upstream
// that depends on it (dependents) and everything downstream it relies on
// (dependencies), merged into one graph with each node tagged by Direction. A
// file or module root expands to the symbols it contains so its whole footprint
// is traced, not just the container node.
func (s *Store) ImpactMap(ctx context.Context, projectID, rootID string, depth, limit int) (model.Graph, error) {
	root, err := s.Entity(ctx, projectID, rootID)
	if err != nil {
		return model.Graph{}, err
	}
	seeds, err := s.impactSeeds(ctx, projectID, root)
	if err != nil {
		return model.Graph{}, err
	}
	return s.impactFrom(ctx, projectID, seeds, "root", rootID, depth, limit)
}

// ChangeImpact is the blast radius of a set of already-resolved changed entities
// (e.g. the symbols a working-tree diff touched). Seeds are tagged "changed";
// dependents and dependencies are tagged as for any other impact map.
func (s *Store) ChangeImpact(ctx context.Context, projectID string, seeds []string, depth, limit int) (model.Graph, error) {
	return s.impactFrom(ctx, projectID, seeds, "changed", "", depth, limit)
}

// impactFrom is the shared engine behind ImpactMap and ChangeImpact: walk
// upstream (dependents) and downstream (dependencies) from a seed set, merge, and
// tag every node's Direction. Seed nodes take seedTag; others become dependent /
// dependency / both.
func (s *Store) impactFrom(ctx context.Context, projectID string, seeds []string, seedTag, rootID string, depth, limit int) (model.Graph, error) {
	depth, limit = clampDepth(depth), clampNodes(limit)
	if len(seeds) == 0 {
		return model.Graph{}, ErrNotFound
	}
	seedSet := make(map[string]bool, len(seeds))
	for _, id := range seeds {
		seedSet[id] = true
	}

	dependents, _, err := s.traversalNodes(ctx, projectID, seeds, "upstream", depth, limit, impactEdgeKinds)
	if err != nil {
		return model.Graph{}, err
	}
	dependencies, _, err := s.traversalNodes(ctx, projectID, seeds, "downstream", depth, limit, impactEdgeKinds)
	if err != nil {
		return model.Graph{}, err
	}

	byID := make(map[string]*model.Entity)
	order := make([]string, 0, len(dependents)+len(dependencies))
	add := func(entity model.Entity, direction string) {
		if existing, ok := byID[entity.ID]; ok {
			if existing.Direction != seedTag && existing.Direction != direction {
				existing.Direction = "both"
			}
			if entity.Distance < existing.Distance {
				existing.Distance = entity.Distance
			}
			return
		}
		node := entity
		node.Direction = direction
		if seedSet[node.ID] {
			node.Direction = seedTag
		}
		byID[node.ID] = &node
		order = append(order, node.ID)
	}
	for _, entity := range dependents {
		add(entity, "dependent")
	}
	for _, entity := range dependencies {
		add(entity, "dependency")
	}

	nodes := make([]model.Entity, 0, len(order))
	truncated := false
	for _, id := range order {
		if len(nodes) >= limit {
			truncated = true
			break
		}
		nodes = append(nodes, *byID[id])
	}
	if len(nodes) == 0 {
		return model.Graph{}, ErrNotFound
	}
	edges, err := s.edgesForNodes(ctx, projectID, nodes, impactEdgeKinds)
	return model.Graph{Nodes: nodes, Edges: edges, RootID: rootID, Truncated: truncated, Limit: limit, MaxDepth: depth}, err
}

// EntitiesInRanges returns the ids of entities in `path` whose source range
// overlaps any of the given 1-based inclusive [start,end] line ranges. Structural
// entities with no real range (module/file) never match a positive range.
func (s *Store) EntitiesInRanges(ctx context.Context, projectID, path string, ranges [][2]int) ([]string, error) {
	if len(ranges) == 0 {
		return nil, nil
	}
	conditions := make([]string, 0, len(ranges))
	args := []any{projectID, path}
	for _, r := range ranges {
		conditions = append(conditions, "(e.start_line <= ? AND e.end_line >= ?)")
		args = append(args, r[1], r[0]) // overlap: entity.start <= range.end AND entity.end >= range.start
	}
	query := fmt.Sprintf(`
		SELECT DISTINCT e.id
		FROM entities e
		JOIN projects p ON p.active_run_id = e.run_id
		JOIN files f ON f.id = e.file_id
		WHERE e.project_id = ? AND f.path = ? AND (%s)`,
		strings.Join(conditions, " OR "))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanIDs(rows)
}

// FileEntityID returns the id of the file entity for a repo-relative path, or ""
// if the current graph has no such file (e.g. an ignored or newly deleted file).
func (s *Store) FileEntityID(ctx context.Context, projectID, path string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		SELECT e.id FROM entities e JOIN projects p ON p.active_run_id = e.run_id
		WHERE e.project_id = ? AND e.kind = 'file' AND e.qualified_name = ?`, projectID, path).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// impactSeeds expands a root entity into the set of symbols whose change is
// equivalent to changing the root: a symbol is itself; a file is itself plus the
// symbols it defines; a module is itself plus its files and their symbols.
func (s *Store) impactSeeds(ctx context.Context, projectID string, root *model.Entity) ([]string, error) {
	switch root.Kind {
	case "file":
		symbols, err := s.definedSymbols(ctx, projectID, []string{root.ID})
		if err != nil {
			return nil, err
		}
		return append([]string{root.ID}, symbols...), nil
	case "module", "package":
		files, err := s.containedFiles(ctx, projectID, root.ID)
		if err != nil {
			return nil, err
		}
		symbols, err := s.definedSymbols(ctx, projectID, files)
		if err != nil {
			return nil, err
		}
		return append(append([]string{root.ID}, files...), symbols...), nil
	default:
		return []string{root.ID}, nil
	}
}

func (s *Store) containedFiles(ctx context.Context, projectID, moduleID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.target_entity_id FROM relationships r JOIN projects p ON p.active_run_id=r.run_id
		WHERE r.project_id=? AND r.source_entity_id=? AND r.relationship_type='contains'`, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	return scanIDs(rows)
}

func (s *Store) definedSymbols(ctx context.Context, projectID string, fileIDs []string) ([]string, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT r.target_entity_id FROM relationships r JOIN projects p ON p.active_run_id=r.run_id
		WHERE r.project_id=? AND r.relationship_type='defines' AND r.source_entity_id IN (%s)`,
		placeholders(len(fileIDs)))
	args := append([]any{projectID}, stringArgs(fileIDs)...)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanIDs(rows)
}

// traversalNodes walks the relationship graph from one or more seed entities and
// returns the reached nodes (each carrying its Distance from the nearest seed).
func (s *Store) traversalNodes(ctx context.Context, projectID string, seeds []string, direction string, depth, limit int, kinds []string) ([]model.Entity, bool, error) {
	if len(seeds) == 0 {
		return nil, false, nil
	}
	depth, limit = clampDepth(depth), clampNodes(limit)

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

	seedValues := strings.TrimSuffix(strings.Repeat("(?),", len(seeds)), ",")
	query := fmt.Sprintf(`
		WITH RECURSIVE walk(node_id, depth, path) AS (
			SELECT column1, 0, '/' || column1 || '/' FROM (VALUES %[5]s)
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
		nextExpr, placeholders(len(kinds)), joinPredicate, entityColumns, seedValues)

	args := make([]any, 0, len(seeds)+len(kinds)+3)
	args = append(args, stringArgs(seeds)...)
	args = append(args, projectID)
	args = append(args, stringArgs(kinds)...)
	args = append(args, depth, limit+1, projectID)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("traverse graph: %w", err)
	}
	defer rows.Close()
	nodes := make([]model.Entity, 0, limit)
	truncated := false
	for rows.Next() {
		var e model.Entity
		var metadata []byte
		var isTest int
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.RunID, &e.FileID, &e.Kind, &e.Name, &e.QualifiedName, &e.Language,
			&e.Range.StartLine, &e.Range.StartColumn, &e.Range.EndLine, &e.Range.EndColumn, &isTest, &metadata, &e.Distance); err != nil {
			return nil, false, err
		}
		e.IsTest = isTest != 0
		_ = json.Unmarshal(metadata, &e.Metadata)
		if len(nodes) < limit {
			nodes = append(nodes, e)
		} else {
			truncated = true
		}
	}
	return nodes, truncated, rows.Err()
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
