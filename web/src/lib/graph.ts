import type { Entity, GraphFilters, GraphResponse } from '../types'

const meaningfulKinds = new Set([
  'package',
  'module',
  'file',
  'function',
  'method',
  'class',
  'struct',
  'interface',
  'route',
])

const referenceKinds = new Set(['external_symbol', 'unresolved_symbol'])

export interface FilteredGraph {
  graph?: GraphResponse
  hiddenTests: number
  hiddenReferences: number
  hiddenOther: number
  hiddenTotal: number
}

export function filterGraph(
  graph: GraphResponse | undefined,
  filters: GraphFilters,
  preserveEntityId = '',
): FilteredGraph {
  if (!graph) {
    return { graph: undefined, hiddenTests: 0, hiddenReferences: 0, hiddenOther: 0, hiddenTotal: 0 }
  }

  let hiddenTests = 0
  let hiddenReferences = 0
  let hiddenOther = 0
  const nodes = graph.nodes.filter((node) => {
    if (node.id === preserveEntityId || node.id === graph.rootId) return true
    if (node.isTest && !filters.showTests) {
      hiddenTests++
      return false
    }
    if (referenceKinds.has(node.kind) && !filters.showReferences) {
      hiddenReferences++
      return false
    }
    if (!meaningfulKinds.has(node.kind) && !referenceKinds.has(node.kind)) {
      hiddenOther++
      return false
    }
    return true
  })

  const visibleIds = new Set(nodes.map((node) => node.id))
  const edges = graph.edges.filter(
    (relationship) => visibleIds.has(relationship.source) && visibleIds.has(relationship.target),
  )

  return {
    graph: { ...graph, nodes, edges },
    hiddenTests,
    hiddenReferences,
    hiddenOther,
    hiddenTotal: hiddenTests + hiddenReferences + hiddenOther,
  }
}

export function isMeaningfulEntity(entity: Entity): boolean {
  return meaningfulKinds.has(entity.kind)
}

export function entityKindLabel(kind: string): string {
  return kind.replaceAll('_', ' ')
}

export function graphLayoutKey(graph: GraphResponse | undefined, mode: string): string {
  if (!graph) return ''
  const nodes = graph.nodes.map((node) => node.id).join(',')
  const topology = graph.edges.map((edge) => `${edge.source}>${edge.target}:${edge.kind}`).join(',')
  return `${mode}:${graph.rootId ?? 'overview'}:${nodes}:${topology}`
}
