import { useDeferredValue, useEffect, useMemo, useState } from 'react'
import { Background, Controls, MarkerType, MiniMap, ReactFlow, type Edge } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { GraphResponse } from '../../types'
import { AtlasNode, type AtlasFlowNode } from './AtlasNode'

const nodeTypes = { atlas: AtlasNode }

export function GraphCanvas({
  graph,
  selectedEntityId,
  mode,
  onSelect,
  onExplore,
}: {
  graph?: GraphResponse
  selectedEntityId: string
  mode: string
  onSelect: (id: string) => void
  onExplore: (id: string) => void
}) {
  const deferredGraph = useDeferredValue(graph)
  const layoutKey = useMemo(
    () =>
      deferredGraph
        ? `${deferredGraph.rootId ?? 'overview'}:${deferredGraph.nodes
            .map((node) => node.id)
            .join(',')}:${deferredGraph.edges.length}`
        : '',
    [deferredGraph],
  )
  const [layout, setLayout] = useState<{
    key: string
    positions: Record<string, { x: number; y: number }>
  }>()
  const positions = layout?.key === layoutKey ? layout.positions : undefined

  useEffect(() => {
    if (!deferredGraph?.nodes.length) {
      return
    }

    let settled = false
    const worker = new Worker(new URL('../../layout.worker.ts', import.meta.url), {
      type: 'module',
    })
    const finishLayout = (nextPositions: Record<string, { x: number; y: number }>) => {
      if (settled) {
        return
      }
      settled = true
      setLayout({ key: layoutKey, positions: nextPositions })
    }
    const fallbackTimeout = window.setTimeout(() => {
      finishLayout(fallbackPositions(deferredGraph.nodes.map((node) => node.id)))
      worker.terminate()
    }, 2_500)

    worker.onmessage = (event) => {
      window.clearTimeout(fallbackTimeout)
      finishLayout(event.data as Record<string, { x: number; y: number }>)
    }
    worker.onerror = () => {
      window.clearTimeout(fallbackTimeout)
      finishLayout(fallbackPositions(deferredGraph.nodes.map((node) => node.id)))
    }
    worker.postMessage({ nodes: deferredGraph.nodes, edges: deferredGraph.edges })
    return () => {
      settled = true
      window.clearTimeout(fallbackTimeout)
      worker.terminate()
    }
  }, [deferredGraph, layoutKey])

  const compact = (deferredGraph?.nodes.length ?? 0) > 60
  const showEdgeLabels = (deferredGraph?.nodes.length ?? 0) <= 36

  const nodes = useMemo<AtlasFlowNode[]>(() => {
    if (!deferredGraph || !positions) {
      return []
    }
    return deferredGraph.nodes.map((entity) => ({
      id: entity.id,
      type: 'atlas',
      data: { entity, compact },
      position: positions[entity.id] ?? { x: 0, y: 0 },
      selected: entity.id === selectedEntityId,
    }))
  }, [compact, deferredGraph, positions, selectedEntityId])

  const edges = useMemo<Edge[]>(() => {
    if (!deferredGraph) {
      return []
    }
    return deferredGraph.edges.map((relationship) => {
      const relationshipCount = relationship.metadata?.relationshipCount
      const count = typeof relationshipCount === 'number' ? relationshipCount : 0
      const resolved = relationship.resolution === 'resolved'
      return {
        id: relationship.id,
        source: relationship.source,
        target: relationship.target,
        label: showEdgeLabels
          ? `${relationship.kind.replace('_', ' ')}${count > 1 ? ` ×${count}` : ''}`
          : undefined,
        animated: mode === 'flow' && resolved && deferredGraph.nodes.length < 50,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: resolved ? '#8298b8' : '#56657b',
        },
        style: {
          stroke: resolved ? '#667c9e' : '#4b5b72',
          strokeDasharray: resolved ? undefined : '6 5',
        },
        labelStyle: {
          fill: '#8293aa',
          fontSize: 10,
          fontWeight: 600,
        },
      }
    })
  }, [deferredGraph, mode, showEdgeLabels])

  if (!graph) {
    return <GraphLoading label="Loading graph metadata…" />
  }
  if (!graph.nodes.length) {
    return <GraphEmpty />
  }
  if (!positions) {
    return <GraphLoading label="Arranging this view…" />
  }

  return (
    <ReactFlow
      key={`${graph.rootId ?? 'overview'}:${graph.nodes.length}:${graph.edges.length}`}
      edges={edges}
      elementsSelectable
      fitView
      fitViewOptions={{ maxZoom: 1.05, padding: 0.24 }}
      maxZoom={1.8}
      minZoom={0.18}
      nodes={nodes}
      nodesConnectable={false}
      nodesDraggable={false}
      nodeTypes={nodeTypes}
      onNodeClick={(_, node) => onSelect(node.id)}
      onNodeDoubleClick={(_, node) => {
        const entity = node.data.entity
        if (entity.kind === 'module' || entity.kind === 'file') {
          onExplore(node.id)
        }
      }}
      onlyRenderVisibleElements
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#24334a" gap={30} size={1} />
      <Controls showInteractive={false} />
      {nodes.length <= 80 && (
        <MiniMap maskColor="rgba(7, 12, 21, 0.78)" nodeColor="#3d506b" pannable zoomable />
      )}
    </ReactFlow>
  )
}

function fallbackPositions(ids: string[]): Record<string, { x: number; y: number }> {
  const columns = Math.max(1, Math.ceil(Math.sqrt(ids.length)))
  return Object.fromEntries(
    ids.map((id, index) => [
      id,
      {
        x: (index % columns) * 240,
        y: Math.floor(index / columns) * 120,
      },
    ]),
  )
}

function GraphLoading({ label }: { label: string }) {
  return (
    <div className="graph-state">
      <div className="loader" />
      <p>{label}</p>
    </div>
  )
}

function GraphEmpty() {
  return (
    <div className="graph-state">
      <div className="empty-orbit" aria-hidden="true" />
      <h3>No relationships in this view</h3>
      <p>Choose another symbol or return to the architecture overview.</p>
    </div>
  )
}
