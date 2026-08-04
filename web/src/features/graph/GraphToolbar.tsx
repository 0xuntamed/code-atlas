import { Icon } from '../../components/Icon'
import type { FilteredGraph } from '../../lib/graph'
import type { ArchitectureCrumb, GraphView } from '../../types'

export function GraphToolbar({
  architecturePath,
  filtered,
  view,
  onNavigate,
}: {
  architecturePath: ArchitectureCrumb[]
  filtered: FilteredGraph
  view: GraphView
  onNavigate: (index: number) => void
}) {
  const graph = filtered.graph
  return (
    <header className="flex min-h-14 items-center justify-between gap-4 border-b border-border bg-canvas/60 px-4 py-2 backdrop-blur sm:px-5">
      <div className="min-w-0">
        <span className="text-[9px] font-bold uppercase tracking-[0.18em] text-dim">
          {view === 'architecture' ? 'Architecture explorer' : `${view} analysis`}
        </span>
        <nav
          aria-label="Architecture path"
          className="mt-1 flex min-w-0 items-center gap-1 text-xs"
        >
          <button
            className="font-semibold text-foreground hover:text-primary"
            onClick={() => onNavigate(-1)}
          >
            Overview
          </button>
          {view === 'architecture'
            ? architecturePath.map((crumb, index) => (
                <span className="flex min-w-0 items-center gap-1" key={crumb.id}>
                  <Icon className="size-3 shrink-0 text-dim" name="chevron-right" />
                  <button
                    className="max-w-40 truncate font-medium text-muted hover:text-foreground"
                    onClick={() => onNavigate(index)}
                    title={crumb.name}
                  >
                    {crumb.name}
                  </button>
                </span>
              ))
            : null}
        </nav>
      </div>

      <div className="flex shrink-0 items-center gap-3 text-right">
        {filtered.hiddenTotal > 0 ? (
          <span className="hidden rounded-full border border-border bg-panel px-2.5 py-1 text-[9px] font-semibold text-muted sm:block">
            {filtered.hiddenTotal} noise hidden
          </span>
        ) : null}
        <div>
          <strong className="block font-mono text-[10px] font-semibold text-foreground">
            {graph ? `${graph.nodes.length}/${graph.limit}` : '—'}
          </strong>
          <small className="block text-[8px] uppercase tracking-[0.13em] text-dim">
            node budget
          </small>
        </div>
      </div>
    </header>
  )
}
