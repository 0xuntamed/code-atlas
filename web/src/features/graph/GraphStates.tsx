import type { GraphView } from '../../types'

export function RootRequired({ view }: { view: GraphView }) {
  return (
    <div className="graph-state">
      <div className="empty-orbit" aria-hidden="true" />
      <h3>Select a symbol to begin {view} analysis</h3>
      <p>Use local metadata search, then choose this graph mode again.</p>
    </div>
  )
}

export function GraphError() {
  return (
    <div className="graph-state error-state">
      <h3>The graph could not be loaded</h3>
      <p>The last valid analysis is still available. Try narrowing or refreshing the view.</p>
    </div>
  )
}
