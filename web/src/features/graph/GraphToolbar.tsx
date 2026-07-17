import type { ArchitectureCrumb, GraphResponse, GraphView } from '../../types'

export function GraphToolbar({
  architecturePath,
  graph,
  view,
  onNavigate,
}: {
  architecturePath: ArchitectureCrumb[]
  graph?: GraphResponse
  view: GraphView
  onNavigate: (index: number) => void
}) {
  return (
    <header className="graph-toolbar">
      <div className="graph-title">
        <span>{view === 'architecture' ? 'Architecture explorer' : `${view} analysis`}</span>
        <nav aria-label="Architecture path" className="breadcrumbs">
          <button onClick={() => onNavigate(-1)}>Overview</button>
          {view === 'architecture' &&
            architecturePath.map((crumb, index) => (
              <span key={crumb.id}>
                <i>/</i>
                <button onClick={() => onNavigate(index)}>{crumb.name}</button>
              </span>
            ))}
        </nav>
      </div>
      <div className="graph-health">
        <span className="live-dot" />
        <div>
          <strong>Local derived graph</strong>
          <small>{graph ? `${graph.nodes.length}/${graph.limit} node budget` : 'Loading'}</small>
        </div>
      </div>
    </header>
  )
}
