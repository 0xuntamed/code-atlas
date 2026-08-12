import { useDeferredValue, useEffect, useMemo, useState } from 'react'
import { Background, Controls, MarkerType, MiniMap, Position, ReactFlow } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { graphLayoutKey } from '../../lib/graph'
import { DENSITY } from '../../lib/graphConfig'
import type { GraphResponse, Lens } from '../../types'
import { AtlasNode, type AtlasFlowNode } from './AtlasNode'
import { RelationshipEdge, type RelationshipFlowEdge } from './RelationshipEdge'

const nodeTypes = { atlas: AtlasNode }
const edgeTypes = { relationship: RelationshipEdge }

export function GraphCanvas({
  graph,
  hiddenCount,
  selectedEntityId,
  layoutMode,
  lensActive,
  highlightNodeIds,
  highlightEdgeIds,
  onSelect,
  onExplore,
}: {
  graph?: GraphResponse
  hiddenCount: number
  selectedEntityId: string
  layoutMode: Lens
  lensActive: boolean
  highlightNodeIds: Set<string>
  highlightEdgeIds: Set<string>
  onSelect: (id: string) => void
  onExplore: (id: string) => void
}) {
  const deferredGraph = useDeferredValue(graph)
  const layoutKey = useMemo(
    () => graphLayoutKey(deferredGraph, layoutMode),
    [deferredGraph, layoutMode],
  )
  const [layout, setLayout] = useState<{
    key: string
    positions: Record<string, { x: number; y: number }>
  }>()
  const positions = layout?.key === layoutKey ? layout.positions : undefined

  useEffect(() => {
    if (!deferredGraph?.nodes.length) return

    let settled = false
    const worker = new Worker(new URL('../../layout.worker.ts', import.meta.url), {
      type: 'module',
    })
    const finish = (nextPositions: Record<string, { x: number; y: number }>) => {
      if (settled) return
      settled = true
      setLayout({ key: layoutKey, positions: nextPositions })
    }
    const timeout = window.setTimeout(() => {
      finish(fallbackPositions(deferredGraph.nodes.map((node) => node.id)))
      worker.terminate()
    }, 2_000)

    worker.onmessage = (event) => {
      window.clearTimeout(timeout)
      finish(event.data as Record<string, { x: number; y: number }>)
      worker.terminate()
    }
    worker.onerror = () => {
      window.clearTimeout(timeout)
      finish(fallbackPositions(deferredGraph.nodes.map((node) => node.id)))
      worker.terminate()
    }
    worker.postMessage({
      mode: layoutMode,
      nodes: deferredGraph.nodes.map(({ id }) => ({ id })),
      edges: deferredGraph.edges.map(({ id, source, target }) => ({ id, source, target })),
    })

    return () => {
      settled = true
      window.clearTimeout(timeout)
      worker.terminate()
    }
  }, [deferredGraph, layoutKey, layoutMode])

  const nodeCount = deferredGraph?.nodes.length ?? 0
  const compact = nodeCount > DENSITY.compactAbove
  const showEdgeLabels = nodeCount <= DENSITY.edgeLabelsUpTo
  const direction = layoutMode === 'structure' ? 'horizontal' : 'vertical'

  const nodes = useMemo<AtlasFlowNode[]>(() => {
    if (!deferredGraph || !positions) return []
    return deferredGraph.nodes.map((entity) => {
      const highlighted = highlightNodeIds.has(entity.id)
      return {
        id: entity.id,
        type: 'atlas',
        data: {
          entity,
          compact,
          direction,
          highlighted: lensActive && highlighted,
          dimmed: lensActive && !highlighted && entity.id !== selectedEntityId,
        },
        position: positions[entity.id] ?? { x: 0, y: 0 },
        selected: entity.id === selectedEntityId,
      }
    })
  }, [compact, deferredGraph, direction, highlightNodeIds, lensActive, positions, selectedEntityId])

  const edges = useMemo<RelationshipFlowEdge[]>(() => {
    if (!deferredGraph) return []
    return deferredGraph.edges.map((relationship) => {
      const aggregateCount = relationship.metadata?.relationshipCount
      const count = typeof aggregateCount === 'number' ? aggregateCount : 0
      const resolved = relationship.resolution === 'resolved'
      const onLens = lensActive && highlightEdgeIds.has(relationship.id)
      const faded = lensActive && !onLens
      const stroke = resolved ? 'var(--ui-graph-edge)' : 'var(--ui-graph-edge-muted)'
      return {
        id: relationship.id,
        type: 'relationship',
        source: relationship.source,
        target: relationship.target,
        sourcePosition: direction === 'vertical' ? Position.Bottom : Position.Right,
        targetPosition: direction === 'vertical' ? Position.Top : Position.Left,
        data: {
          label: showEdgeLabels
            ? `${relationship.kind.replaceAll('_', ' ')}${count > 1 ? ` x${count}` : ''}`
            : undefined,
        },
        animated: onLens && resolved && nodeCount < DENSITY.animateBelow,
        markerEnd: { type: MarkerType.ArrowClosed, color: stroke },
        style: {
          stroke,
          strokeWidth: onLens ? 1.8 : resolved ? 1.4 : 1,
          strokeDasharray: resolved ? undefined : '6 5',
          opacity: faded ? 0.25 : 1,
        },
      }
    })
  }, [deferredGraph, direction, highlightEdgeIds, lensActive, nodeCount, showEdgeLabels])

  if (!graph) return <GraphLoading label="Loading derived metadata..." />
  if (!graph.nodes.length) return <GraphEmpty hiddenCount={hiddenCount} />
  if (!positions) return <GraphLoading label="Arranging focused nodes..." />

  return (
    <ReactFlow
      aria-label={`${layoutMode} graph`}
      edgeTypes={edgeTypes}
      edges={edges}
      elementsSelectable
      fitView
      fitViewOptions={{ maxZoom: 1.08, padding: 0.28 }}
      maxZoom={1.8}
      minZoom={0.15}
      nodeTypes={nodeTypes}
      nodes={nodes}
      nodesConnectable={false}
      nodesDraggable={false}
      onNodeClick={(_, node) => onSelect(node.id)}
      onNodeDoubleClick={(_, node) => {
        const entity = node.data.entity
        if (entity.kind === 'module' || entity.kind === 'file') onExplore(node.id)
      }}
      onlyRenderVisibleElements
      proOptions={{ hideAttribution: true }}
    >
      <Background color="var(--ui-graph-grid)" gap={28} size={1} />
      <Controls showInteractive={false} />
      {nodes.length <= DENSITY.minimapUpTo ? (
        <MiniMap
          maskColor="var(--ui-graph-mask)"
          nodeColor="var(--ui-graph-node)"
          pannable
          zoomable
        />
      ) : null}
    </ReactFlow>
  )
}

function fallbackPositions(ids: string[]): Record<string, { x: number; y: number }> {
  const columns = Math.max(1, Math.ceil(Math.sqrt(ids.length)))
  return Object.fromEntries(
    ids.map((id, index) => [
      id,
      { x: (index % columns) * 260, y: Math.floor(index / columns) * 120 },
    ]),
  )
}

function GraphLoading({ label }: { label: string }) {
  return (
    <div className="grid h-full place-items-center">
      <div className="text-center">
        <span className="mx-auto block size-7 animate-spin-slow rounded-full border-2 border-border-strong border-t-primary" />
        <p className="mt-4 text-xs font-semibold text-muted">{label}</p>
      </div>
    </div>
  )
}

function GraphEmpty({ hiddenCount }: { hiddenCount: number }) {
  return (
    <div className="grid h-full place-items-center px-6 text-center">
      <div className="max-w-sm">
        <div className="mx-auto grid size-14 place-items-center rounded-2xl border border-border bg-panel text-dim">
          <span className="size-2 rounded-full bg-primary" />
        </div>
        <h3 className="mt-5 text-sm font-semibold text-foreground">
          No focused relationships here
        </h3>
        <p className="mt-2 text-xs leading-5 text-muted">
          {hiddenCount > 0
            ? `${hiddenCount} low-signal nodes are hidden. Adjust graph filters to include them.`
            : 'Choose another symbol or return to the architecture overview.'}
        </p>
      </div>
    </div>
  )
}
