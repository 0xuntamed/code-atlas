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

// useWorkspaceGraph owns the workspace's server state: an always-on structural
// base map, plus the blast radius of whatever node is selected — merged onto the
// base map as a color-coded overlay. Selecting a node is all it takes; there is
// no mode to toggle.
export function useWorkspaceGraph(project: Project): WorkspaceGraph {
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

    // Union the blast radius onto the base map. Nodes keep their base position as
    // context; overlay-only nodes join if their direction passes the filter.
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

    // Keep only direction tags whose node survived filtering, and collect the
    // edges that run inside the blast radius.
    const impactEdgeIds = new Set<string>()
    if (filtered.graph) {
      const visible = new Set(filtered.graph.nodes.map((node) => node.id))
      for (const id of [...directionById.keys()]) if (!visible.has(id)) directionById.delete(id)
      for (const edge of filtered.graph.edges) {
        if (directionById.has(edge.source) && directionById.has(edge.target)) {
          impactEdgeIds.add(edge.id)
        }
      }
    }

    return {
      displayGraph: filtered.graph,
      hiddenTotal: filtered.hiddenTotal,
      directionById,
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
    impactQuery.data,
    impactFilter,
    selectedEntity,
    selectedEntityId,
    showReferences,
    showTests,
  ])
}
