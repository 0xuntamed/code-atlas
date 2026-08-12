import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { isTraceable } from '../../lib/entityKinds'
import { filterGraph } from '../../lib/graph'
import { useAtlasStore } from '../../store'
import type { Entity, GraphResponse, Project } from '../../types'

export interface WorkspaceGraph {
  displayGraph?: GraphResponse
  hiddenTotal: number
  highlightNodeIds: Set<string>
  highlightEdgeIds: Set<string>
  // True only when an overlay is actually loaded and merged onto the base map.
  lensActive: boolean
  selectedEntity?: Entity
  isError: boolean
  isLoading: boolean
}

// useWorkspaceGraph owns the workspace's server state: one always-on structural
// base map, plus a selection-driven flow/impact overlay merged onto it as a
// highlight layer. Splitting this out keeps the workspace component presentational.
export function useWorkspaceGraph(project: Project): WorkspaceGraph {
  const selectedEntityId = useAtlasStore((state) => state.selectedEntityId)
  const lens = useAtlasStore((state) => state.lens)
  const scopePath = useAtlasStore((state) => state.scopePath)
  const showTests = useAtlasStore((state) => state.showTests)
  const showReferences = useAtlasStore((state) => state.showReferences)

  const scopeId = scopePath.at(-1)?.id ?? ''
  const baseQuery = useQuery({
    queryKey: ['graph', project.id, project.activeRunId, 'structure', scopeId],
    queryFn: ({ signal }) => api.architecture(project.id, scopeId, signal),
  })

  // Resolve the selected entity: prefer one already present on the base map,
  // otherwise fetch it (e.g. a symbol chosen from search that isn't on-canvas).
  const entityInBase = baseQuery.data?.nodes.find((node) => node.id === selectedEntityId)
  const entityQuery = useQuery({
    queryKey: ['entity', project.id, selectedEntityId],
    queryFn: ({ signal }) => api.entity(project.id, selectedEntityId, signal),
    enabled: Boolean(selectedEntityId) && !entityInBase,
  })
  const selectedEntity = entityInBase ?? entityQuery.data

  const traceable = isTraceable(selectedEntity?.kind)
  const overlayRequested = lens !== 'structure' && traceable && Boolean(selectedEntityId)

  const overlayQuery = useQuery({
    queryKey: ['graph', project.id, project.activeRunId, lens, selectedEntityId],
    queryFn: ({ signal }) =>
      lens === 'flow'
        ? api.flow(project.id, selectedEntityId, signal)
        : api.impact(project.id, selectedEntityId, signal),
    enabled: overlayRequested,
  })

  return useMemo(() => {
    const base = baseQuery.data
    const overlay = overlayRequested ? overlayQuery.data : undefined
    const highlightNodeIds = new Set<string>()
    const highlightEdgeIds = new Set<string>()

    // Union the overlay onto the base map; the overlay's members become the
    // highlighted subgraph, everything else stays as dimmed context.
    let merged = base
    if (base && overlay) {
      const nodeById = new Map(base.nodes.map((node) => [node.id, node]))
      for (const node of overlay.nodes) {
        nodeById.set(node.id, node)
        highlightNodeIds.add(node.id)
      }
      const edgeById = new Map(base.edges.map((edge) => [edge.id, edge]))
      for (const edge of overlay.edges) {
        edgeById.set(edge.id, edge)
        highlightEdgeIds.add(edge.id)
      }
      merged = {
        ...base,
        nodes: [...nodeById.values()],
        edges: [...edgeById.values()],
        rootId: overlay.rootId ?? base.rootId,
      }
    } else if (overlay) {
      merged = overlay
      overlay.nodes.forEach((node) => highlightNodeIds.add(node.id))
      overlay.edges.forEach((edge) => highlightEdgeIds.add(edge.id))
    }

    const lensActive = Boolean(overlay)
    const filtered = filterGraph(
      merged,
      { showTests, showReferences: showReferences || lensActive },
      selectedEntityId,
    )

    // A highlight is meaningful only if its node/edge survived filtering.
    if (filtered.graph) {
      const visibleNodes = new Set(filtered.graph.nodes.map((node) => node.id))
      for (const id of [...highlightNodeIds]) if (!visibleNodes.has(id)) highlightNodeIds.delete(id)
      const visibleEdges = new Set(filtered.graph.edges.map((edge) => edge.id))
      for (const id of [...highlightEdgeIds]) if (!visibleEdges.has(id)) highlightEdgeIds.delete(id)
    }

    return {
      displayGraph: filtered.graph,
      hiddenTotal: filtered.hiddenTotal,
      highlightNodeIds,
      highlightEdgeIds,
      lensActive,
      selectedEntity,
      isError: baseQuery.isError,
      isLoading: baseQuery.isLoading,
    }
  }, [
    baseQuery.data,
    baseQuery.isError,
    baseQuery.isLoading,
    overlayQuery.data,
    overlayRequested,
    selectedEntity,
    selectedEntityId,
    showReferences,
    showTests,
  ])
}
