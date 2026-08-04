import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { Button } from '../../components/Button'
import { Dialog } from '../../components/Dialog'
import { Icon } from '../../components/Icon'

export function SkippedFiles({ projectId }: { projectId: string }) {
  const [open, setOpen] = useState(false)
  const query = useQuery({
    queryKey: ['files', projectId, 'skipped'],
    queryFn: ({ signal }) => api.files(projectId, 'skipped', 500, signal),
    enabled: open,
  })
  const files = query.data?.files ?? []

  return (
    <>
      <Button
        className="w-full justify-between"
        intent="ghost"
        onClick={() => setOpen(true)}
        size="sm"
      >
        <span className="inline-flex items-center gap-2">
          <Icon className="size-3.5" name="shield" />
          Skipped inventory
        </span>
        <Icon className="size-3.5" name="chevron-right" />
      </Button>

      <Dialog
        description="Privacy exclusions, generated output, dependencies, binaries, and unsupported files stay outside the graph."
        onClose={() => setOpen(false)}
        open={open}
        size="lg"
        title="Skipped file inventory"
      >
        <div className="border-b border-border bg-canvas-raised/45 px-5 py-3 text-xs text-muted sm:px-6">
          {query.isLoading
            ? 'Loading local metadata…'
            : query.isError
              ? 'Inventory unavailable'
              : `${files.length} bounded inventory records`}
        </div>
        <div className="max-h-[62dvh] overflow-y-auto p-3 sm:p-4">
          {query.isError ? (
            <div
              className="m-2 rounded-xl border border-danger/30 bg-danger/10 p-5 text-center"
              role="alert"
            >
              <p className="text-sm text-danger-foreground">
                The local API could not load the skipped inventory.
              </p>
              <Button
                className="mt-4"
                intent="outline"
                onClick={() => void query.refetch()}
                size="sm"
              >
                <Icon className="size-3.5" name="refresh" />
                Try again
              </Button>
            </div>
          ) : null}
          {files.map((file) => (
            <article
              className="content-auto flex items-start gap-3 rounded-lg px-3 py-3 hover:bg-surface/65"
              key={file.id}
            >
              <span className="grid size-8 shrink-0 place-items-center rounded-lg border border-border bg-canvas-raised text-dim">
                <Icon className="size-3.5" name={file.isDirectory ? 'folder' : 'file'} />
              </span>
              <div className="min-w-0 flex-1">
                <strong className="block break-all font-mono text-[11px] font-medium text-foreground">
                  {file.path}
                </strong>
                <small className="mt-1 block text-[10px] leading-4 text-muted">
                  {file.ignoreReason || 'Not part of the source graph'}
                </small>
              </div>
              <span className="rounded border border-border px-1.5 py-0.5 text-[8px] font-bold uppercase tracking-wider text-dim">
                {file.isDirectory ? 'dir' : 'file'}
              </span>
            </article>
          ))}
          {!query.isLoading && !query.isError && files.length === 0 ? (
            <p className="py-12 text-center text-sm text-muted">No skipped files were recorded.</p>
          ) : null}
        </div>
      </Dialog>
    </>
  )
}
