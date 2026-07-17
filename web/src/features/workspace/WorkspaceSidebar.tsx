import type { GraphResponse, GraphView } from '../../types'
import { SkippedFiles } from '../files/SkippedFiles'
import { SymbolSearch } from '../search/SymbolSearch'

const viewDescriptions: Record<GraphView, string> = {
  architecture: 'Progressive repository structure',
  flow: 'Downstream execution from a root',
  impact: 'Upstream and downstream dependents',
}

export function WorkspaceSidebar({
  projectId,
  activeView,
  graph,
  onSelect,
  onViewChange,
}: {
  projectId: string
  activeView: GraphView
  graph?: GraphResponse
  onSelect: (id: string) => void
  onViewChange: (view: GraphView) => void
}) {
  return (
    <aside className="left-rail">
      <SymbolSearch projectId={projectId} onSelect={onSelect} />
      <GraphModeNavigation activeView={activeView} onChange={onViewChange} />
      <GraphBudget graph={graph} view={activeView} />
      <SkippedFiles projectId={projectId} />
    </aside>
  )
}

function GraphModeNavigation({
  activeView,
  onChange,
}: {
  activeView: GraphView
  onChange: (view: GraphView) => void
}) {
  const views: GraphView[] = ['architecture', 'flow', 'impact']
  return (
    <nav className="mode-navigation" aria-label="Graph modes">
      <span className="rail-label">Graph mode</span>
      {views.map((view, index) => (
        <button
          className={view === activeView ? 'active' : ''}
          key={view}
          onClick={() => onChange(view)}
        >
          <span className="mode-index">0{index + 1}</span>
          <span>
            <strong>{view}</strong>
            <small>{viewDescriptions[view]}</small>
          </span>
        </button>
      ))}
    </nav>
  )
}

function GraphBudget({ graph, view }: { graph?: GraphResponse; view: GraphView }) {
  const message = graph?.truncated
    ? `View capped at ${graph.limit} nodes. Drill into a narrower scope.`
    : graph
      ? `${graph.nodes.length} nodes · ${graph.edges.length} relationships`
      : view === 'architecture'
        ? 'Loading the module overview.'
        : 'Select a symbol to establish a root.'

  return (
    <section className="graph-budget">
      <span>Render budget</span>
      <strong>{message}</strong>
      <p>Only visible cards mount in the graph canvas.</p>
    </section>
  )
}
