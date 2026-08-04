import { useRef } from 'react'
import { Brand } from '../../components/Brand'
import { Button } from '../../components/Button'
import { Icon } from '../../components/Icon'
import type { Project } from '../../types'

export function ProjectHeader({
  projects,
  selectedProject,
  reanalyzing,
  onAdd,
  onChange,
  onReanalyze,
  onRemove,
  onShutdown,
}: {
  projects: Project[]
  selectedProject?: Project
  reanalyzing: boolean
  onAdd: () => void
  onChange: (id: string) => void
  onReanalyze: (id: string) => void
  onRemove: (project: Project) => void
  onShutdown: () => void
}) {
  const menuRef = useRef<HTMLDetailsElement>(null)
  const closeMenu = () => {
    if (menuRef.current) menuRef.current.open = false
  }

  return (
    <header className="relative z-30 flex h-16 items-center gap-4 border-b border-border bg-canvas/90 px-4 backdrop-blur-xl sm:px-5">
      <div className="hidden shrink-0 sm:block">
        <Brand />
      </div>
      <div className="sm:hidden">
        <Brand compact />
      </div>

      {projects.length > 0 ? (
        <div className="relative min-w-0 flex-1 sm:max-w-xs">
          <label className="sr-only" htmlFor="project-select">
            Repository
          </label>
          <Icon
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-dim"
            name="repository"
          />
          <select
            className="h-10 w-full appearance-none truncate rounded-control border border-border-strong bg-panel pl-9 pr-9 text-sm font-semibold text-foreground hover:border-primary/30"
            id="project-select"
            onChange={(event) => onChange(event.target.value)}
            value={selectedProject?.id ?? ''}
          >
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name}
              </option>
            ))}
          </select>
          <Icon
            className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-dim"
            name="chevron-down"
          />
        </div>
      ) : (
        <span className="flex-1 text-xs font-semibold text-dim">127.0.0.1 · local only</span>
      )}

      <div className="ml-auto flex items-center gap-2">
        {selectedProject ? <StatusBadge status={selectedProject.status} /> : null}
        {selectedProject ? (
          <Button
            aria-label="Reanalyze repository"
            className="hidden lg:inline-flex"
            disabled={reanalyzing}
            intent="outline"
            onClick={() => onReanalyze(selectedProject.id)}
            size="sm"
          >
            <Icon className={`size-3.5 ${reanalyzing ? 'animate-spin' : ''}`} name="refresh" />
            {reanalyzing ? 'Queued…' : 'Reanalyze'}
          </Button>
        ) : null}
        <Button aria-label="Add repository" intent="secondary" onClick={onAdd} size="sm">
          <Icon className="size-3.5" name="add" />
          <span className="hidden md:inline">Add repository</span>
        </Button>

        <details className="relative" ref={menuRef}>
          <summary className="grid size-9 cursor-pointer list-none place-items-center rounded-control border border-border-strong bg-panel-raised text-muted transition-colors hover:bg-surface hover:text-foreground [&::-webkit-details-marker]:hidden">
            <Icon className="size-4" name="menu" />
            <span className="sr-only">Application actions</span>
          </summary>
          <div className="absolute right-0 top-11 w-64 overflow-hidden rounded-xl border border-border-strong bg-panel p-1.5 shadow-[0_22px_60px_var(--ui-shadow)]">
            {selectedProject ? (
              <>
                <button
                  className="flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-muted hover:bg-surface hover:text-foreground disabled:pointer-events-none disabled:opacity-45 lg:hidden"
                  disabled={reanalyzing}
                  onClick={() => {
                    onReanalyze(selectedProject.id)
                    closeMenu()
                  }}
                  type="button"
                >
                  <Icon className="mt-0.5 size-4" name="refresh" />
                  <span>
                    <strong className="block text-xs">
                      {reanalyzing ? 'Analysis queued…' : 'Reanalyze repository'}
                    </strong>
                    <small className="mt-0.5 block text-[10px] text-dim">
                      Refresh derived metadata
                    </small>
                  </span>
                </button>
                <button
                  className="flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-muted hover:bg-danger/10 hover:text-red-200"
                  onClick={() => {
                    onRemove(selectedProject)
                    closeMenu()
                  }}
                  type="button"
                >
                  <Icon className="mt-0.5 size-4" name="trash" />
                  <span>
                    <strong className="block text-xs">Remove repository</strong>
                    <small className="mt-0.5 block text-[10px] text-dim">
                      Delete derived metadata
                    </small>
                  </span>
                </button>
              </>
            ) : null}
            <div className="my-1 border-t border-border" />
            <button
              className="flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-muted hover:bg-danger/10 hover:text-red-200"
              onClick={() => {
                onShutdown()
                closeMenu()
              }}
              type="button"
            >
              <Icon className="mt-0.5 size-4" name="power" />
              <span>
                <strong className="block text-xs">Stop CodeAtlas</strong>
                <small className="mt-0.5 block text-[10px] text-dim">
                  Exit the local Go server
                </small>
              </span>
            </button>
          </div>
        </details>
      </div>
    </header>
  )
}

function StatusBadge({ status }: { status: Project['status'] }) {
  const tones = {
    ready: 'border-primary/25 bg-primary/8 text-primary',
    queued: 'border-warning/25 bg-warning/8 text-amber-200',
    analyzing: 'border-accent/25 bg-accent/8 text-blue-200',
    failed: 'border-danger/25 bg-danger/8 text-red-200',
  } as const
  return (
    <span
      className={`hidden items-center gap-1.5 rounded-full border px-2.5 py-1 text-[9px] font-bold uppercase tracking-[0.13em] sm:inline-flex ${tones[status]}`}
    >
      <i className="size-1.5 rounded-full bg-current" />
      {status}
    </span>
  )
}
