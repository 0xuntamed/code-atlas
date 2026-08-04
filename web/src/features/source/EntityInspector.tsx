import { lazy, Suspense, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, APIError } from '../../api'
import { Button } from '../../components/Button'
import { Icon } from '../../components/Icon'
import { entityKindLabel } from '../../lib/graph'
import type { Entity, GraphView, SourceEvidence } from '../../types'

const LocalSourceEditor = lazy(() => import('./LocalSourceEditor'))
const preloadSourceEditor = () => void import('./LocalSourceEditor')

export function EntityInspector({
  projectId,
  entity,
  onClose,
  onExplore,
  onViewChange,
}: {
  projectId: string
  entity?: Entity
  onClose: () => void
  onExplore: (entity: Entity) => void
  onViewChange: (view: GraphView) => void
}) {
  const [sourceOpen, setSourceOpen] = useState(false)
  const source = useQuery({
    queryKey: ['source', projectId, entity?.id],
    queryFn: ({ signal }) => api.source(projectId, entity!.id, signal),
    enabled: sourceOpen && Boolean(entity?.fileId),
    retry: false,
  })

  if (!entity) {
    return (
      <aside className="hidden min-h-0 flex-col border-l border-border bg-canvas-raised/72 xl:flex">
        <div className="grid flex-1 place-items-center px-8 text-center">
          <div>
            <span className="mx-auto grid size-12 place-items-center rounded-2xl border border-border bg-panel text-dim">
              <Icon className="size-5" name="code" />
            </span>
            <h3 className="mt-4 text-xs font-semibold text-foreground">Select a focused node</h3>
            <p className="mt-2 text-[10px] leading-5 text-muted">
              Inspect identity, range, graph actions, and verified local source.
            </p>
          </div>
        </div>
      </aside>
    )
  }

  const expandable = entity.kind === 'module' || entity.kind === 'file'
  return (
    <aside className="absolute inset-y-0 right-0 z-20 flex w-full max-w-[390px] flex-col border-l border-border bg-canvas-raised shadow-[-24px_0_70px_var(--ui-shadow)] xl:relative xl:inset-auto xl:z-auto xl:w-auto xl:max-w-none xl:shadow-none">
      <header className="flex items-start justify-between gap-4 border-b border-border px-4 py-4">
        <div className="min-w-0">
          <span className="text-[9px] font-bold uppercase tracking-[0.17em] text-primary">
            {entityKindLabel(entity.kind)}
          </span>
          <h2
            className="mt-1 truncate text-base font-semibold tracking-tight text-foreground"
            title={entity.name}
          >
            {entity.name}
          </h2>
        </div>
        <Button aria-label="Close inspector" intent="ghost" onClick={onClose} size="icon">
          <Icon className="size-4" name="close" />
        </Button>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="border-b border-border px-4 py-4">
          <p className="break-all font-mono text-[10px] leading-5 text-muted">
            {entity.qualifiedName}
          </p>
          <div className="mt-4 grid grid-cols-2 gap-2">
            <Fact label="Language" value={entity.language || 'derived'} />
            <Fact label="Source range" value={sourceRange(entity)} />
            <Fact label="Graph distance" value={entity.distance?.toString() ?? 'root'} />
            <Fact label="Test code" value={entity.isTest ? 'included' : 'no'} />
          </div>
        </div>

        <section className="border-b border-border px-4 py-4">
          <span className="text-[9px] font-bold uppercase tracking-[0.17em] text-dim">Actions</span>
          <div className="mt-3 grid grid-cols-2 gap-2">
            {expandable ? (
              <Button className="col-span-2" onClick={() => onExplore(entity)} size="sm">
                <Icon className="size-3.5" name="architecture" />
                Open contents
              </Button>
            ) : (
              <>
                <Button intent="secondary" onClick={() => onViewChange('flow')} size="sm">
                  <Icon className="size-3.5" name="flow" />
                  Trace flow
                </Button>
                <Button intent="secondary" onClick={() => onViewChange('impact')} size="sm">
                  <Icon className="size-3.5" name="impact" />
                  Check impact
                </Button>
              </>
            )}
            {entity.fileId ? (
              <Button
                className="col-span-2"
                intent="outline"
                onClick={() => setSourceOpen((current) => !current)}
                onFocus={preloadSourceEditor}
                onMouseEnter={preloadSourceEditor}
                size="sm"
              >
                <Icon className="size-3.5" name="code" />
                {sourceOpen ? 'Hide source evidence' : 'Load source evidence'}
              </Button>
            ) : null}
          </div>
        </section>

        {sourceOpen ? (
          <SourcePanel source={source.data} loading={source.isLoading} error={source.error} />
        ) : null}
      </div>
    </aside>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-panel px-3 py-2.5">
      <span className="block text-[8px] font-bold uppercase tracking-[0.12em] text-dim">
        {label}
      </span>
      <strong
        className="mt-1 block truncate text-[10px] font-semibold text-foreground"
        title={value}
      >
        {value}
      </strong>
    </div>
  )
}

function SourcePanel({
  source,
  loading,
  error,
}: {
  source?: SourceEvidence
  loading: boolean
  error: Error | null
}) {
  if (loading) {
    return <div className="p-6 text-center text-xs text-muted">Verifying local source hash…</div>
  }
  if (error) {
    const stale = error instanceof APIError && error.code === 'source_stale'
    return (
      <div className="m-4 rounded-xl border border-warning/25 bg-warning/5 p-4 text-xs leading-5 text-warning-foreground">
        {stale
          ? 'Source changed after analysis. Reanalyze the repository to refresh this evidence.'
          : 'No verified source evidence is available for this node.'}
      </div>
    )
  }
  if (!source) return null

  return (
    <section>
      <header className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
        <span className="min-w-0 truncate font-mono text-[9px] text-muted" title={source.filePath}>
          {source.filePath}
        </span>
        <small className="shrink-0 font-mono text-[8px] text-dim">
          L{source.startLine}–{source.endLine}
        </small>
      </header>
      <div className="h-80 bg-[#1e1e1e]">
        <Suspense
          fallback={
            <div className="grid h-full place-items-center text-xs text-muted">
              Loading local editor…
            </div>
          }
        >
          <LocalSourceEditor language={editorLanguage(source.language)} value={source.code} />
        </Suspense>
      </div>
    </section>
  )
}

function sourceRange(entity: Entity): string {
  if (!entity.range.startLine) return '—'
  return `${entity.range.startLine}–${entity.range.endLine || entity.range.startLine}`
}

function editorLanguage(language: string): string {
  if (language === 'go' || language === 'python' || language === 'javascript') return language
  return 'typescript'
}
