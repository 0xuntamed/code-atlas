import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { filterGraph } from '../../lib/graph'
import { useAtlasStore } from '../../store'
import type { Entity, GraphResponse, ImpactDirection, ImpactFilter, Project } from '../../types'

export interface WorkspaceGraph {
  displayGraph?: GraphResponse
  hiddenTotal: number
  // Blast-radius role per node id (root / dependent / dependency / both).
  directionById: Map<string, ImpactDirection>
  // Edge ids that run between two nodes in the blast radius.
  impactEdgeIds: Set<string>
  // True when a selection's blast radius is loaded and overlaid.
  impactActive: boolean
  selectedEntity?: Entity
  isError: boolean
  isLoading: boolean
}

function allowed(direction: ImpactDirection, filter: ImpactFilter): boolean {
  if (direction === 'root' || direction === 'both') return true
  if (filter === 'both') return true
  if (filter === 'dependents') return direction === 'dependent'
  return direction === 'dependency'
}

// The file a symbol belongs to, taken from the qualified name prefix
// ("src/orders.ts::fetchOrders" → "src/orders.ts").
function fileKey(entity: Entity): string | undefined {
  const marker = entity.qualifiedName.indexOf('::')
  return marker > 0 ? entity.qualifiedName.slice(0, marker) : undefined
}

function aggregateDirection(members: Entity[], directionById: Map<string, ImpactDirection>) {
  let dependent = false
  let dependency = false
  for (const member of members) {
    const direction = directionById.get(member.id)
    if (direction === 'dependent') dependent = true
    else if (direction === 'dependency') dependency = true
    else if (direction === 'both') {
      dependent = true
      dependency = true
    }
  }
  if (dependent && dependency) return 'both' as const
  return dependent ? ('dependent' as const) : ('dependency' as const)
}

// rollupByFile collapses each file's affected symbols into one "N affected" file
// node, so the blast radius reads at file level by default. Expanding a file key
// reveals its individual symbols again.
function rollupByFile(
  graph: GraphResponse,
  directionById: Map<string, ImpactDirection>,
  rootId: string,
  expanded: Set<string>,
): { graph: GraphResponse; directionById: Map<string, ImpactDirection> } {
  const groupable = (node: Entity) =>
    directionById.has(node.id) &&
    node.id !== rootId &&
    node.kind !== 'file' &&
    node.kind !== 'module' &&
    node.kind !== 'package' &&
    Boolean(fileKey(node))

  const members = new Map<string, Entity[]>()
  for (const node of graph.nodes) {
    if (!groupable(node)) continue
    const key = fileKey(node) as string
    const list = members.get(key) ?? []
    list.push(node)
    members.set(key, list)
  }

  const hidden = new Set<string>()
  const groupNodes: Entity[] = []
  const nextDirection = new Map(directionById)
  for (const [key, list] of members) {
    if (list.length < 2 || expanded.has(key)) continue
    const direction = aggregateDirection(list, directionById)
    for (const member of list) {
      hidden.add(member.id)
      nextDirection.delete(member.id)
    }
    const id = `group:${key}`
    groupNodes.push({
      ...list[0],
      id,
      kind: 'file',
      name: key.split('/').pop() ?? key,
      qualifiedName: key,
      fileId: undefined,
      direction,
      metadata: { group: true, groupKey: key, affectedCount: list.length },
    })
    nextDirection.set(id, direction)
  }

  if (groupNodes.length === 0) return { graph, directionById }

  const nodes = graph.nodes.filter((node) => !hidden.has(node.id)).concat(groupNodes)
  const edges = graph.edges.filter((edge) => !hidden.has(edge.source) && !hidden.has(edge.target))
  return { graph: { ...graph, nodes, edges }, directionById: nextDirection }
}

// useWorkspaceGraph owns the workspace's server state: an always-on structural
// base map, plus the blast radius of whatever node is selected — merged onto the
// base map as a color-coded overlay, rolled up to file level by default.
export function useWorkspaceGraph(project: Project, expandedFiles: Set<string>): WorkspaceGraph {
  const selectedEntityId = useAtlasStore((state) => state.selectedEntityId)
  const impactFilter = useAtlasStore((state) => state.impactFilter)
  const scopePath = useAtlasStore((state) => state.scopePath)
  const showTests = useAtlasStore((state) => state.showTests)
  const showReferences = useAtlasStore((state) => state.showReferences)

  const scopeId = scopePath.at(-1)?.id ?? ''
  const baseQuery = useQuery({
    queryKey: ['graph', project.id, project.activeRunId, 'structure', scopeId],
    queryFn: ({ signal }) => api.architecture(project.id, scopeId, signal),
  })

  const entityInBase = baseQuery.data?.nodes.find((node) => node.id === selectedEntityId)
  const entityQuery = useQuery({
    queryKey: ['entity', project.id, selectedEntityId],
    queryFn: ({ signal }) => api.entity(project.id, selectedEntityId, signal),
    enabled: Boolean(selectedEntityId) && !entityInBase,
  })
  const selectedEntity = entityInBase ?? entityQuery.data

  const impactQuery = useQuery({
    queryKey: ['impact-map', project.id, project.activeRunId, selectedEntityId],
    queryFn: ({ signal }) => api.impactMap(project.id, selectedEntityId, signal),
    enabled: Boolean(selectedEntityId),
  })

  return useMemo(() => {
    const base = baseQuery.data
    const impact = selectedEntityId ? impactQuery.data : undefined
    const directionById = new Map<string, ImpactDirection>()

    let merged = base
    if (base && impact) {
      const nodeById = new Map(base.nodes.map((node) => [node.id, node]))
      const edgeById = new Map(base.edges.map((edge) => [edge.id, edge]))
      for (const node of impact.nodes) {
        const direction = (node.direction ?? 'dependency') as ImpactDirection
        const pass = allowed(direction, impactFilter)
        if (nodeById.has(node.id) || pass) nodeById.set(node.id, node)
        if (pass) directionById.set(node.id, direction)
      }
      for (const edge of impact.edges) edgeById.set(edge.id, edge)
      merged = {
        ...base,
        nodes: [...nodeById.values()],
        edges: [...edgeById.values()],
        rootId: impact.rootId ?? base.rootId,
      }
    } else if (impact) {
      merged = impact
      for (const node of impact.nodes) {
        const direction = (node.direction ?? 'dependency') as ImpactDirection
        if (allowed(direction, impactFilter)) directionById.set(node.id, direction)
      }
    }

    const impactActive = Boolean(impact) && directionById.size > 0
    const filtered = filterGraph(
      merged,
      { showTests, showReferences: showReferences || impactActive },
      selectedEntityId,
    )

    let displayGraph = filtered.graph
    let dirMap = directionById
    if (displayGraph) {
      const visible = new Set(displayGraph.nodes.map((node) => node.id))
      for (const id of [...dirMap.keys()]) if (!visible.has(id)) dirMap.delete(id)
      if (impactActive) {
        const rolled = rollupByFile(displayGraph, dirMap, selectedEntityId, expandedFiles)
        displayGraph = rolled.graph
        dirMap = rolled.directionById
      }
    }

    const impactEdgeIds = new Set<string>()
    if (displayGraph) {
      for (const edge of displayGraph.edges) {
        if (dirMap.has(edge.source) && dirMap.has(edge.target)) impactEdgeIds.add(edge.id)
      }
    }

    return {
      displayGraph,
      hiddenTotal: filtered.hiddenTotal,
      directionById: dirMap,
      impactEdgeIds,
      impactActive,
      selectedEntity,
      isError: baseQuery.isError,
      isLoading: baseQuery.isLoading,
    }
  }, [
    baseQuery.data,
    baseQuery.isError,
    baseQuery.isLoading,
    expandedFiles,
    impactQuery.data,
    impactFilter,
    selectedEntity,
    selectedEntityId,
    showReferences,
    showTests,
  ])
}
