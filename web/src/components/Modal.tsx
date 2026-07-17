import { useEffect, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

export function Modal({
  open,
  title,
  description,
  children,
  onClose,
}: {
  open: boolean
  title: string
  description?: string
  children: ReactNode
  onClose: () => void
}) {
  useEffect(() => {
    if (!open) {
      return
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose, open])

  if (!open) {
    return null
  }

  return createPortal(
    <div className="modal-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        aria-describedby={description ? 'modal-description' : undefined}
        aria-labelledby="modal-title"
        aria-modal="true"
        className="modal-card"
        onMouseDown={(event) => event.stopPropagation()}
        role="dialog"
      >
        <header className="modal-header">
          <div>
            <h2 id="modal-title">{title}</h2>
            {description && <p id="modal-description">{description}</p>}
          </div>
          <button aria-label="Close dialog" className="icon-button" onClick={onClose}>
            ×
          </button>
        </header>
        {children}
      </section>
    </div>,
    document.body,
  )
}
