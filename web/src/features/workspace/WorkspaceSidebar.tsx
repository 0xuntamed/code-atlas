import { Icon } from '../../components/Icon'
import { ImpactLegend } from '../graph/ImpactLegend'
import type { GraphResponse, ImpactFilter } from '../../types'
import { SkippedFiles } from '../files/SkippedFiles'
import { SymbolSearch } from '../search/SymbolSearch'

const impactFilters: { value: ImpactFilter; label: string }[] = [
  { value: 'both', label: 'Both' },
  { value: 'dependents', label: 'Breaks' },
  { value: 'dependencies', label: 'Relies on' },
]

export function WorkspaceSidebar({
  projectId,
  impactFilter,
  impactActive,
  expandedCount,
  graph,
  hiddenTotal,
  showTests,
  showReferences,
  onSelect,
  onToggleTests,
  onToggleReferences,
  onImpactFilterChange,
  onCollapseFiles,
}: {
  projectId: string
  impactFilter: ImpactFilter
  impactActive: boolean
  expandedCount: number
  graph?: GraphResponse
  hiddenTotal: number
  showTests: boolean
  showReferences: boolean
  onSelect: (id: string) => void
  onToggleTests: () => void
  onToggleReferences: () => void
  onImpactFilterChange: (filter: ImpactFilter) => void
  onCollapseFiles: () => void
}) {
  return (
    <aside className="min-h-0 overflow-y-auto border-b border-border bg-canvas-raised/72 p-3 md:border-b-0 md:border-r md:p-4">
      <SymbolSearch projectId={projectId} showTests={showTests} onSelect={onSelect} />

      <section className="mt-4">
        <span className="px-1 text-[9px] font-bold uppercase tracking-[0.18em] text-dim">
          Blast radius
        </span>
        {impactActive ? (
          <>
            <div
              className="mt-2 grid grid-cols-3 gap-1 rounded-xl border border-border bg-panel p-1"
              role="group"
              aria-label="Blast radius side"
            >
              {impactFilters.map((filter) => {
                const active = impactFilter === filter.value
                return (
                  <button
                    aria-pressed={active}
                    className={`rounded-lg px-2 py-1.5 text-[10px] font-semibold transition-colors ${active ? 'bg-primary/15 text-primary' : 'text-muted hover:text-foreground'}`}
                    key={filter.value}
                    onClick={() => onImpactFilterChange(filter.value)}
                    type="button"
                  >
                    {filter.label}
                  </button>
                )
              })}
            </div>
            <ImpactLegend className="mt-3 px-1" />
            {expandedCount > 0 ? (
              <button
                className="mt-3 w-full rounded-lg border border-border bg-panel px-2 py-1.5 text-[10px] font-semibold text-muted transition-colors hover:text-foreground"
                onClick={onCollapseFiles}
                type="button"
              >
                Collapse {expandedCount} expanded file{expandedCount > 1 ? 's' : ''}
              </button>
            ) : (
              <p className="mt-2 px-1 text-[9px] leading-4 text-dim">
                Affected files group their symbols — click a file node to expand.
              </p>
            )}
          </>
        ) : (
          <p className="mt-2 px-1 text-[10px] leading-4 text-muted">
            Click any node — a file, module, or function — to light up what it changes and what it
            depends on.
          </p>
        )}
      </section>

      <section className="mt-5 hidden md:block">
        <div className="flex items-center justify-between px-1">
          <span className="text-[9px] font-bold uppercase tracking-[0.18em] text-dim">
            Signal filters
          </span>
          <Icon className="size-3.5 text-dim" name="filter" />
        </div>
        <div className="mt-2 space-y-1.5">
          <FilterToggle
            checked={showTests}
            copy="Test files and symbols"
            icon="test"
            label="Include tests"
            onChange={onToggleTests}
          />
          <FilterToggle
            checked={showReferences}
            copy="External and unresolved"
            icon="external"
            label="Reference noise"
            onChange={onToggleReferences}
          />
        </div>
      </section>

      <section className="mt-5 hidden rounded-xl border border-border bg-panel p-3 md:block">
        <div className="flex items-center justify-between">
          <span className="text-[9px] font-bold uppercase tracking-[0.16em] text-dim">
            Focused view
          </span>
          <span className="size-1.5 rounded-full bg-primary" />
        </div>
        <strong className="mt-2 block font-mono text-sm font-semibold text-foreground">
          {graph ? `${graph.nodes.length} nodes` : 'Loading'}
        </strong>
        <p className="mt-1 text-[10px] leading-4 text-muted">
          {graph ? `${graph.edges.length} visible relationships` : 'Reading graph metadata'}
        </p>
        {hiddenTotal > 0 ? (
          <p className="mt-2 border-t border-border pt-2 text-[9px] leading-4 text-dim">
            {hiddenTotal} low-signal nodes hidden by default
          </p>
        ) : null}
      </section>

      <div className="mt-4 hidden border-t border-border pt-3 md:block">
        <SkippedFiles projectId={projectId} />
      </div>
    </aside>
  )
}

function FilterToggle({
  checked,
  label,
  copy,
  icon,
  onChange,
}: {
  checked: boolean
  label: string
  copy: string
  icon: 'test' | 'external'
  onChange: () => void
}) {
  return (
    <button
      aria-checked={checked}
      className="flex w-full items-center gap-3 rounded-xl border border-transparent px-2 py-2 text-left hover:border-border hover:bg-surface"
      onClick={onChange}
      role="switch"
      type="button"
    >
      <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-panel text-dim">
        <Icon className="size-3.5" name={icon} />
      </span>
      <span className="min-w-0 flex-1">
        <strong className="block text-[11px] font-semibold text-foreground">{label}</strong>
        <small className="block text-[9px] text-dim">{copy}</small>
      </span>
      <span
        className={`relative h-5 w-9 rounded-full transition-colors ${checked ? 'bg-primary' : 'bg-surface'}`}
      >
        <i
          className={`absolute top-1/2 size-3.5 -translate-y-1/2 rounded-full bg-foreground shadow transition-transform ${checked ? 'translate-x-[18px]' : 'translate-x-[3px]'}`}
        />
      </span>
    </button>
  )
}
