import { describe, expect, it } from 'vitest'
import { filterGraph, graphLayoutKey } from './lib/graph'
import type { Entity, GraphResponse } from './types'

const entity = (id: string, kind: string, isTest = false): Entity => ({
  id,
  projectId: 'project',
  runId: 'run',
  kind,
  name: id,
  qualifiedName: id,
  range: { startLine: 1, startColumn: 1, endLine: 1, endColumn: 2 },
  isTest,
})

const graph: GraphResponse = {
  nodes: [
    entity('root', 'function'),
    entity('test', 'function', true),
    entity('external', 'external_symbol'),
    entity('unknown', 'made_up_kind'),
  ],
  edges: [
    {
      id: 'root-test',
      projectId: 'project',
      runId: 'run',
      source: 'root',
      target: 'test',
      kind: 'calls',
      confidence: 1,
      range: { startLine: 1, startColumn: 1, endLine: 1, endColumn: 2 },
      resolution: 'resolved',
    },
    {
      id: 'root-external',
      projectId: 'project',
      runId: 'run',
      source: 'root',
      target: 'external',
      kind: 'calls',
      confidence: 0.5,
      range: { startLine: 1, startColumn: 1, endLine: 1, endColumn: 2 },
      resolution: 'external',
    },
  ],
  rootId: 'root',
  truncated: false,
  limit: 120,
}

describe('focused graph filtering', () => {
  it('hides tests, reference noise, and unknown entity kinds by default', () => {
    const filtered = filterGraph(graph, { showTests: false, showReferences: false })
    expect(filtered.graph?.nodes.map((node) => node.id)).toEqual(['root'])
    expect(filtered.graph?.edges).toEqual([])
    expect(filtered.hiddenTotal).toBe(3)
  })

  it('keeps opt-in test and reference nodes with their valid edges', () => {
    const filtered = filterGraph(graph, { showTests: true, showReferences: true })
    expect(filtered.graph?.nodes.map((node) => node.id)).toEqual(['root', 'test', 'external'])
    expect(filtered.graph?.edges.map((edge) => edge.id)).toEqual(['root-test', 'root-external'])
  })

  it('preserves an explicitly selected test root', () => {
    const filtered = filterGraph(graph, { showTests: false, showReferences: false }, 'test')
    expect(filtered.graph?.nodes.map((node) => node.id)).toEqual(['root', 'test'])
  })

  it('hides test-only modules flagged by architecture metadata', () => {
    const testModuleGraph = { ...graph, nodes: [entity('tests', 'module', true)], edges: [] }
    const filtered = filterGraph(testModuleGraph, { showTests: false, showReferences: false })
    expect(filtered.graph?.nodes).toEqual([])
    expect(filtered.hiddenTests).toBe(1)
  })

  it('changes the layout key when edge topology changes at the same size', () => {
    const first = graphLayoutKey(graph, 'flow')
    const rewired = {
      ...graph,
      edges: graph.edges.map((edge, index) =>
        index === 0 ? { ...edge, target: 'external' } : edge,
      ),
    }
    expect(graphLayoutKey(rewired, 'flow')).not.toBe(first)
  })
})
