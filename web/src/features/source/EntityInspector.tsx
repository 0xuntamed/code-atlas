import { lazy, Suspense } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, APIError } from '../../api'
import type { Entity, GraphView } from '../../types'

const LocalSourceEditor = lazy(() => import('./LocalSourceEditor'))

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
  const source = useQuery({
    queryKey: ['source', projectId, entity?.id],
    queryFn: () => api.source(projectId, entity!.id),
    enabled: Boolean(entity?.fileId),
    retry: false,
  })

  if (!entity) {
    return (
      <aside className="inspector empty-inspector">
        <div className="inspector-orbit" aria-hidden="true" />
        <h3>Select a graph node</h3>
        <p>Inspect its exact location, confidence evidence, and local source context.</p>
      </aside>
    )
  }

  const expandable = entity.kind === 'module' || entity.kind === 'file'

  return (
    <aside className="inspector">
      <header className="inspector-header">
        <div>
          <span className="entity-kind">{entity.kind.replace('_', ' ')}</span>
          <h2>{entity.name}</h2>
        </div>
        <button aria-label="Close inspector" className="icon-button" onClick={onClose}>
          ×
        </button>
      </header>

      <p className="qualified-name">{entity.qualifiedName}</p>

      <div className="entity-facts">
        <Fact label="Language" value={entity.language || 'derived'} />
        <Fact label="Range" value={sourceRange(entity)} />
        <Fact label="Distance" value={entity.distance?.toString() ?? 'root'} />
        <Fact label="Test" value={entity.isTest ? 'yes' : 'no'} />
      </div>

      <div className="inspector-actions">
        {expandable && (
          <button className="button primary" onClick={() => onExplore(entity)}>
            Open contents
          </button>
        )}
        {entity.kind !== 'module' && entity.kind !== 'file' && (
          <>
            <button className="button secondary" onClick={() => onViewChange('flow')}>
              Trace flow
            </button>
            <button className="button secondary" onClick={() => onViewChange('impact')}>
              Check impact
            </button>
          </>
        )}
      </div>

      <SourceContent source={source} />
    </aside>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function SourceContent({
  source,
}: {
  source: ReturnType<typeof useQuery<Awaited<ReturnType<typeof api.source>>, Error>>
}) {
  if (source.isError) {
    const stale = source.error instanceof APIError && source.error.code === 'source_stale'
    return (
      <div className="source-message">
        {stale
          ? 'Source changed after analysis. Reanalyze to refresh this evidence.'
          : 'No live source evidence is available for this node.'}
      </div>
    )
  }
  if (!source.data) {
    return null
  }

  return (
    <section className="source-view">
      <header>
        <span>{source.data.filePath}</span>
        <small>
          lines {source.data.startLine}–{source.data.endLine}
        </small>
      </header>
      <Suspense fallback={<div className="source-loading">Loading local editor…</div>}>
        <LocalSourceEditor
          language={editorLanguage(source.data.language)}
          value={source.data.code}
        />
      </Suspense>
    </section>
  )
}

function sourceRange(entity: Entity): string {
  if (!entity.range.startLine) {
    return '—'
  }
  return `${entity.range.startLine}–${entity.range.endLine || entity.range.startLine}`
}

function editorLanguage(language: string): string {
  if (language === 'go' || language === 'python') {
    return language
  }
  return language === 'javascript' ? 'javascript' : 'typescript'
}
