import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { Icon } from '../../components/Icon'
import { useDebouncedValue } from '../../hooks/useDebouncedValue'
import { entityKindLabel, isMeaningfulEntity } from '../../lib/graph'

export function SymbolSearch({
  projectId,
  showTests,
  onSelect,
}: {
  projectId: string
  showTests: boolean
  onSelect: (id: string) => void
}) {
  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query.trim(), 180)
  const search = useQuery({
    queryKey: ['search', projectId, debouncedQuery],
    queryFn: ({ signal }) => api.search(projectId, debouncedQuery, signal),
    enabled: debouncedQuery.length >= 2,
  })

  const results = (search.data?.entities ?? []).filter(
    (entity) => isMeaningfulEntity(entity) && (showTests || !entity.isTest),
  )
  const open = query.trim().length >= 2

  return (
    <div className="relative">
      <label className="sr-only" htmlFor="symbol-search">
        Search symbols and routes
      </label>
      <Icon
        className="pointer-events-none absolute left-3 top-1/2 z-10 size-4 -translate-y-1/2 text-dim"
        name="search"
      />
      <input
        autoComplete="off"
        className="h-10 w-full rounded-control border border-border-strong bg-canvas-raised pl-9 pr-9 text-xs text-foreground placeholder:text-dim focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-primary/10"
        id="symbol-search"
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Find symbols or routes"
        spellCheck={false}
        value={query}
      />
      {query ? (
        <button
          aria-label="Clear search"
          className="absolute right-2 top-1/2 z-10 grid size-6 -translate-y-1/2 place-items-center rounded text-dim hover:bg-surface hover:text-foreground"
          onClick={() => setQuery('')}
          type="button"
        >
          <Icon className="size-3.5" name="close" />
        </button>
      ) : null}

      {open ? (
        <div className="absolute inset-x-0 top-12 z-40 max-h-80 overflow-y-auto rounded-xl border border-border-strong bg-panel p-1.5 shadow-[0_20px_60px_var(--ui-shadow)]">
          {search.isFetching ? (
            <p className="px-3 py-4 text-center text-[11px] text-muted">
              Searching local metadata…
            </p>
          ) : null}
          {search.isError ? (
            <p className="px-3 py-4 text-center text-[11px] text-danger-foreground" role="alert">
              Search is temporarily unavailable.
            </p>
          ) : null}
          {!search.isFetching && !search.isError && results.length === 0 ? (
            <p className="px-3 py-4 text-center text-[11px] text-muted">
              No focused symbols found.
            </p>
          ) : null}
          {results.map((entity) => (
            <button
              className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left hover:bg-surface"
              key={entity.id}
              onClick={() => {
                onSelect(entity.id)
                setQuery('')
              }}
              type="button"
            >
              <span className="grid size-7 shrink-0 place-items-center rounded-lg border border-border bg-canvas-raised text-dim">
                <Icon className="size-3.5" name={entity.kind === 'route' ? 'route' : 'code'} />
              </span>
              <span className="min-w-0 flex-1">
                <strong className="block truncate text-xs font-semibold text-foreground">
                  {entity.name}
                </strong>
                <small className="mt-0.5 block truncate font-mono text-[9px] text-dim">
                  {entity.qualifiedName}
                </small>
              </span>
              <em className="text-[8px] font-bold uppercase not-italic tracking-[0.12em] text-muted">
                {entityKindLabel(entity.kind)}
              </em>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
}
