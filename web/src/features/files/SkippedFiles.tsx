import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'

const visibleFileLimit = 100

export function SkippedFiles({ projectId }: { projectId: string }) {
  const [open, setOpen] = useState(false)
  const query = useQuery({
    queryKey: ['files', projectId, 'skipped'],
    queryFn: () => api.files(projectId, 'skipped', 200),
    enabled: open,
  })
  const files = query.data?.files ?? []
  const visibleFiles = files.slice(0, visibleFileLimit)

  return (
    <section className={`skipped-files ${open ? 'is-open' : ''}`}>
      <button className="skipped-trigger" onClick={() => setOpen((current) => !current)}>
        <span>
          <strong>Skipped inventory</strong>
          <small>Privacy and noise exclusions</small>
        </span>
        <b aria-hidden="true">{open ? '−' : '+'}</b>
      </button>

      {open && (
        <div className="skipped-content">
          {query.isLoading && <p>Loading metadata…</p>}
          {visibleFiles.map((file) => (
            <div className="skipped-row" key={file.id}>
              <span>{file.isDirectory ? 'DIR' : 'FILE'}</span>
              <div>
                <strong>{file.path}</strong>
                <small>{file.ignoreReason}</small>
              </div>
            </div>
          ))}
          {files.length > visibleFileLimit && (
            <p className="bounded-list-note">
              Showing the first {visibleFileLimit} records. Use the API to inspect the full
              inventory.
            </p>
          )}
          {!query.isLoading && files.length === 0 && <p>No skipped files were recorded.</p>}
        </div>
      )}
    </section>
  )
}
