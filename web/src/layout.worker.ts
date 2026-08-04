/// <reference lib="webworker" />
import ELK from 'elkjs/lib/elk.bundled.js'

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
      'elk.direction': mode === 'architecture' ? 'RIGHT' : 'DOWN',
      'elk.spacing.nodeNode': '44',
      'elk.layered.spacing.nodeNodeBetweenLayers': '92',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
      'elk.layered.cycleBreaking.strategy': 'GREEDY',
    },
    children: nodes.map((node) => ({ id: node.id, width: 214, height: 82 })),
    edges: edges.map((edge) => ({ id: edge.id, sources: [edge.source], targets: [edge.target] })),
  })
  self.postMessage(
    Object.fromEntries(
      (graph.children ?? []).map((node) => [node.id, { x: node.x ?? 0, y: node.y ?? 0 }]),
    ),
  )
}
