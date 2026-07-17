/// <reference lib="webworker" />
import ELK from 'elkjs/lib/elk.bundled.js'

const elk = new ELK()

self.onmessage = async (
  event: MessageEvent<{
    nodes: { id: string }[]
    edges: { id: string; source: string; target: string }[]
  }>,
) => {
  const { nodes, edges } = event.data
  const graph = await elk.layout({
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'RIGHT',
      'elk.spacing.nodeNode': '44',
      'elk.layered.spacing.nodeNodeBetweenLayers': '90',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
    },
    children: nodes.map((node) => ({ id: node.id, width: 190, height: 76 })),
    edges: edges.map((edge) => ({ id: edge.id, sources: [edge.source], targets: [edge.target] })),
  })
  const positions = Object.fromEntries(
    (graph.children ?? []).map((node) => [node.id, { x: node.x ?? 0, y: node.y ?? 0 }]),
  )
  self.postMessage(positions)
}
