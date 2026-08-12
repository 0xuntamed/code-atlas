import { Icon } from '../../components/Icon'
import type { ArchitectureCrumb, GraphResponse } from '../../types'
import { ImpactLegend } from './ImpactLegend'

export function GraphToolbar({
  scopePath,
  graph,
  hiddenTotal,
  impactActive,
  selectedName,
  onNavigate,
}: {
  scopePath: ArchitectureCrumb[]
  graph?: GraphResponse
  hiddenTotal: number
  impactActive: boolean
  selectedName?: string
  onNavigate: (index: number) => void
}) {
  return (
    <header className="flex min-h-14 items-center justify-between gap-4 border-b border-border bg-canvas/60 px-4 py-2 backdrop-blur sm:px-5">
      <div className="min-w-0">
        <span className="text-[9px] font-bold uppercase tracking-[0.18em] text-dim">
          {impactActive ? 'Blast radius' : 'Architecture explorer'}
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
          {scopePath.map((crumb, index) => (
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
          ))}
        </nav>
        {impactActive && selectedName ? (
          <span className="mt-1.5 inline-flex items-center gap-1 text-[9px] font-medium text-dim">
            <Icon className="size-3 text-primary" name="impact" />
            Ripple from <strong className="font-semibold text-muted">{selectedName}</strong>
          </span>
        ) : null}
      </div>

      <div className="flex shrink-0 items-center gap-3 text-right">
        {impactActive ? <ImpactLegend className="hidden lg:flex" /> : null}
        {hiddenTotal > 0 ? (
          <span className="hidden rounded-full border border-border bg-panel px-2.5 py-1 text-[9px] font-semibold text-muted sm:block">
            {hiddenTotal} noise hidden
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
