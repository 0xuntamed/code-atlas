import { useEffect, useRef, useState } from 'react'
import { Logo } from '../../components/Logo'
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
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!menuOpen) {
      return
    }
    const closeMenu = (event: MouseEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', closeMenu)
    return () => document.removeEventListener('mousedown', closeMenu)
  }, [menuOpen])

  return (
    <header className="app-header">
      <Logo />

      {projects.length > 0 ? (
        <div className="project-switcher">
          <label htmlFor="project-select">Repository</label>
          <select
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
        </div>
      ) : (
        <span className="local-pill">127.0.0.1 · local only</span>
      )}

      <div className="header-actions">
        <button className="button add-project" onClick={onAdd}>
          <span aria-hidden="true">+</span>
          Add repository
        </button>

        {selectedProject && (
          <>
            <span className={`project-status ${selectedProject.status}`}>
              {selectedProject.status}
            </span>
            <button
              className="button secondary"
              disabled={reanalyzing}
              onClick={() => onReanalyze(selectedProject.id)}
            >
              {reanalyzing ? 'Queued…' : 'Reanalyze'}
            </button>
          </>
        )}

        <div className="project-menu" ref={menuRef}>
          <button
            aria-expanded={menuOpen}
            aria-label="Application actions"
            className="icon-button menu-trigger"
            onClick={() => setMenuOpen((current) => !current)}
          >
            •••
          </button>
          {menuOpen && (
            <div className="menu-popover">
              {selectedProject && (
                <button
                  onClick={() => {
                    onRemove(selectedProject)
                    setMenuOpen(false)
                  }}
                >
                  <span>Remove repository</span>
                  <small>Delete derived metadata</small>
                </button>
              )}
              <button
                className="danger-action"
                onClick={() => {
                  onShutdown()
                  setMenuOpen(false)
                }}
              >
                <span>Stop CodeAtlas</span>
                <small>Exit the local server</small>
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
