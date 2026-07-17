import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { useDebouncedValue } from '../../hooks/useDebouncedValue'

export function SymbolSearch({
  projectId,
  onSelect,
}: {
  projectId: string
  onSelect: (id: string) => void
}) {
  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query.trim(), 180)
  const search = useQuery({
    queryKey: ['search', projectId, debouncedQuery],
    queryFn: () => api.search(projectId, debouncedQuery),
    enabled: debouncedQuery.length > 1,
  })

  const results = search.data?.entities ?? []
  const open = query.trim().length > 1

  return (
    <div className="symbol-search">
      <div className="search-input">
        <span aria-hidden="true" />
        <input
          aria-label="Search symbols and routes"
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search symbols or routes"
          value={query}
        />
        {query && (
          <button aria-label="Clear search" onClick={() => setQuery('')}>
            ×
          </button>
        )}
      </div>

      {open && (
        <div className="search-results">
          {search.isFetching && <p className="search-status">Searching local metadata…</p>}
          {!search.isFetching && results.length === 0 && (
            <p className="search-status">No symbols found.</p>
          )}
          {results.map((entity) => (
            <button
              key={entity.id}
              onClick={() => {
                onSelect(entity.id)
                setQuery('')
              }}
            >
              <i className={`kind-dot ${entity.kind}`} />
              <span>
                <strong>{entity.name}</strong>
                <small>{entity.qualifiedName}</small>
              </span>
              <em>{entity.kind.replace('_', ' ')}</em>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
