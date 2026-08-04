import type { ReactNode } from 'react'
import { Button } from './Button'
import { Dialog } from './Dialog'

export function ConfirmDialog({
  open,
  title,
  confirmLabel,
  danger = false,
  busy = false,
  error,
  children,
  onCancel,
  onConfirm,
}: {
  open: boolean
  title: string
  confirmLabel: string
  danger?: boolean
  busy?: boolean
  error?: string
  children: ReactNode
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <Dialog open={open} title={title} onClose={onCancel} size="sm">
      <div className="px-5 py-5 text-sm leading-6 text-muted sm:px-6">
        {children}
        {error ? (
          <p
            className="mt-4 rounded-lg border border-danger/35 bg-danger/10 px-3 py-2.5 text-danger-foreground"
            role="alert"
          >
            {error}
          </p>
        ) : null}
      </div>
      <footer className="flex justify-end gap-3 border-t border-border bg-canvas-raised/45 px-5 py-4 sm:px-6">
        <Button disabled={busy} intent="ghost" onClick={onCancel}>
          Cancel
        </Button>
        <Button disabled={busy} intent={danger ? 'danger' : 'primary'} onClick={onConfirm}>
          {busy ? 'Working…' : confirmLabel}
        </Button>
      </footer>
    </Dialog>
  )
}
