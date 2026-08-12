import { Icon, type IconName } from '../../components/Icon'
import type { GraphResponse, Lens } from '../../types'
import { SkippedFiles } from '../files/SkippedFiles'
import { SymbolSearch } from '../search/SymbolSearch'

const lensContent: Record<Lens, { icon: IconName; label: string; copy: string }> = {
  structure: { icon: 'architecture', label: 'Structure', copy: 'Modules and symbols' },
  flow: { icon: 'flow', label: 'Flow', copy: 'Downstream execution' },
  impact: { icon: 'impact', label: 'Impact', copy: 'Change blast radius' },
}

export function WorkspaceSidebar({
  projectId,
  activeLens,
  lensEnabled,
  graph,
  hiddenTotal,
  showTests,
  showReferences,
  onSelect,
  onToggleTests,
  onToggleReferences,
  onLensChange,
}: {
  projectId: string
  activeLens: Lens
  // Whether the overlay lenses can be applied (a traceable symbol is selected).
  lensEnabled: boolean
  graph?: GraphResponse
  hiddenTotal: number
  showTests: boolean
  showReferences: boolean
  onSelect: (id: string) => void
  onToggleTests: () => void
  onToggleReferences: () => void
  onLensChange: (lens: Lens) => void
}) {
  return (
    <aside className="min-h-0 overflow-y-auto border-b border-border bg-canvas-raised/72 p-3 md:border-b-0 md:border-r md:p-4">
      <SymbolSearch projectId={projectId} showTests={showTests} onSelect={onSelect} />

      <section className="mt-4">
        <span className="px-1 text-[9px] font-bold uppercase tracking-[0.18em] text-dim">Lens</span>
        <nav aria-label="Graph lens" className="mt-2 grid grid-cols-3 gap-1.5 md:grid-cols-1">
          {(Object.keys(lensContent) as Lens[]).map((lens) => {
            const item = lensContent[lens]
            const active = activeLens === lens
            const disabled = lens !== 'structure' && !lensEnabled
            return (
              <button
                aria-current={active ? 'page' : undefined}
                className={`flex min-w-0 items-center gap-3 rounded-xl border px-2.5 py-2.5 text-left transition-colors md:px-3 ${active ? 'border-primary/30 bg-primary/8 text-foreground' : 'border-transparent text-muted hover:border-border hover:bg-surface hover:text-foreground'} ${disabled ? 'cursor-not-allowed opacity-40 hover:border-transparent hover:bg-transparent' : ''}`}
                disabled={disabled}
                key={lens}
                onClick={() => onLensChange(lens)}
                title={
                  disabled ? 'Select a function, method, or route to use this lens' : undefined
                }
                type="button"
              >
                <span
                  className={`grid size-8 shrink-0 place-items-center rounded-lg ${active ? 'bg-primary/12 text-primary' : 'bg-panel text-dim'}`}
                >
                  <Icon className="size-4" name={item.icon} />
                </span>
                <span className="hidden min-w-0 md:block">
                  <strong className="block text-xs font-semibold">{item.label}</strong>
                  <small className="mt-0.5 block truncate text-[9px] text-dim">{item.copy}</small>
                </span>
              </button>
            )
          })}
        </nav>
        {!lensEnabled ? (
          <p className="mt-2 hidden px-1 text-[9px] leading-4 text-dim md:block">
            Select a function, method, or route to trace its flow or impact on the map.
          </p>
        ) : null}
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
