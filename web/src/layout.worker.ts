/// <reference lib="webworker" />
import ELK from 'elkjs/lib/elk.bundled.js'
import { NODE_HEIGHT, NODE_WIDTH } from './lib/graphConfig'

const elk = new ELK()

self.onmessage = async (
  event: MessageEvent<{
    mode: string
    nodes: { id: string }[]
    edges: { id: string; source: string; target: string }[]
  }>,
) => {
  const { mode, nodes, edges } = event.data
  const graph = await elk.layout({
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      // Structure map reads left-to-right; the flow/impact lenses read top-down.
      'elk.direction': mode === 'structure' ? 'RIGHT' : 'DOWN',
      'elk.spacing.nodeNode': '44',
      'elk.layered.spacing.nodeNodeBetweenLayers': '92',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
      'elk.layered.cycleBreaking.strategy': 'GREEDY',
    },
    children: nodes.map((node) => ({ id: node.id, width: NODE_WIDTH, height: NODE_HEIGHT })),
    edges: edges.map((edge) => ({ id: edge.id, sources: [edge.source], targets: [edge.target] })),
  })
  self.postMessage(
    Object.fromEntries(
      (graph.children ?? []).map((node) => [node.id, { x: node.x ?? 0, y: node.y ?? 0 }]),
    ),
  )
}
