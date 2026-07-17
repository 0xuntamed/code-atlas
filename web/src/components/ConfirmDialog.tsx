import type { ReactNode } from 'react'
import { Modal } from './Modal'

export function ConfirmDialog({
  open,
  title,
  confirmLabel,
  danger = false,
  busy = false,
  children,
  onCancel,
  onConfirm,
}: {
  open: boolean
  title: string
  confirmLabel: string
  danger?: boolean
  busy?: boolean
  children: ReactNode
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <Modal open={open} title={title} onClose={onCancel}>
      <div className="confirm-copy">{children}</div>
      <footer className="modal-actions">
        <button className="button secondary" disabled={busy} onClick={onCancel}>
          Cancel
        </button>
        <button
          className={`button ${danger ? 'danger' : 'primary'}`}
          disabled={busy}
          onClick={onConfirm}
        >
          {busy ? 'Working…' : confirmLabel}
        </button>
      </footer>
    </Modal>
  )
}
