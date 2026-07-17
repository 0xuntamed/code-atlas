import { useState, type FormEvent } from 'react'
import type { SourceType } from '../../types'

export interface ProjectInput {
  type: SourceType
  path?: string
  url?: string
  ref?: string
}

export function ProjectForm({
  busy,
  error,
  compact = false,
  onSubmit,
}: {
  busy: boolean
  error?: string
  compact?: boolean
  onSubmit: (source: ProjectInput) => void
}) {
  const [type, setType] = useState<SourceType>('local')
  const [value, setValue] = useState('')
  const [gitRef, setGitRef] = useState('')

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedValue = value.trim()
    if (!trimmedValue) {
      return
    }
    onSubmit(
      type === 'local'
        ? { type, path: trimmedValue }
        : { type, url: trimmedValue, ref: gitRef.trim() || undefined },
    )
  }

  return (
    <form className={`project-form ${compact ? 'compact' : ''}`} onSubmit={submit}>
      <div className="source-toggle" aria-label="Repository source">
        <button
          aria-pressed={type === 'local'}
          className={type === 'local' ? 'active' : ''}
          onClick={() => setType('local')}
          type="button"
        >
          Local folder
        </button>
        <button
          aria-pressed={type === 'git'}
          className={type === 'git' ? 'active' : ''}
          onClick={() => setType('git')}
          type="button"
        >
          Git repository
        </button>
      </div>

      <label>
        <span>{type === 'local' ? 'Absolute repository path' : 'HTTPS or SSH Git URL'}</span>
        <input
          autoFocus
          onChange={(event) => setValue(event.target.value)}
          placeholder={
            type === 'local' ? 'C:\\work\\payments-api' : 'https://github.com/acme/payments-api.git'
          }
          spellCheck={false}
          value={value}
        />
      </label>

      {type === 'git' && (
        <label>
          <span>
            Branch or tag <small>optional</small>
          </span>
          <input
            onChange={(event) => setGitRef(event.target.value)}
            placeholder="main"
            spellCheck={false}
            value={gitRef}
          />
        </label>
      )}

      {error && <div className="inline-error">{error}</div>}

      <button
        className="button primary submit-project"
        disabled={!value.trim() || busy}
        type="submit"
      >
        {busy ? 'Registering repository…' : 'Build local atlas'}
        <span aria-hidden="true">→</span>
      </button>
    </form>
  )
}
