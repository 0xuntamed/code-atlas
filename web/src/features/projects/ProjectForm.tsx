import { useId, useState, type FormEvent } from 'react'
import { Button } from '../../components/Button'
import { Icon } from '../../components/Icon'
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
  onSubmit,
}: {
  busy: boolean
  error?: string
  onSubmit: (source: ProjectInput) => void
}) {
  const [type, setType] = useState<SourceType>('local')
  const [value, setValue] = useState('')
  const [gitRef, setGitRef] = useState('')
  const valueId = useId()
  const refId = useId()

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedValue = value.trim()
    if (!trimmedValue) return
    onSubmit(
      type === 'local'
        ? { type, path: trimmedValue }
        : { type, url: trimmedValue, ref: gitRef.trim() || undefined },
    )
  }

  return (
    <form className="p-5 sm:p-6" onSubmit={submit}>
      <div
        aria-label="Repository source"
        className="grid grid-cols-2 gap-1 rounded-xl border border-border bg-canvas-raised p-1"
      >
        <SourceOption
          active={type === 'local'}
          icon="folder"
          label="Local folder"
          onClick={() => setType('local')}
        />
        <SourceOption
          active={type === 'git'}
          icon="repository"
          label="Git repository"
          onClick={() => setType('git')}
        />
      </div>

      <label className="mt-5 block" htmlFor={valueId}>
        <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.14em] text-muted">
          {type === 'local' ? 'Absolute repository path' : 'HTTPS or SSH Git URL'}
        </span>
        <input
          autoFocus
          className="h-11 w-full rounded-control border border-border-strong bg-canvas-raised px-3.5 font-mono text-sm text-foreground placeholder:text-dim focus:border-primary/60 focus:outline-none focus:ring-2 focus:ring-primary/15"
          id={valueId}
          onChange={(event) => setValue(event.target.value)}
          placeholder={
            type === 'local' ? 'C:\\work\\payments-api' : 'https://github.com/acme/payments-api.git'
          }
          spellCheck={false}
          value={value}
        />
      </label>

      {type === 'git' ? (
        <label className="mt-4 block" htmlFor={refId}>
          <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.14em] text-muted">
            Branch or tag <small className="normal-case tracking-normal text-dim">optional</small>
          </span>
          <input
            className="h-11 w-full rounded-control border border-border-strong bg-canvas-raised px-3.5 font-mono text-sm text-foreground placeholder:text-dim focus:border-primary/60 focus:outline-none focus:ring-2 focus:ring-primary/15"
            id={refId}
            onChange={(event) => setGitRef(event.target.value)}
            placeholder="main"
            spellCheck={false}
            value={gitRef}
          />
        </label>
      ) : null}

      {error ? (
        <div
          className="mt-4 flex items-start gap-2 rounded-lg border border-danger/35 bg-danger/10 px-3 py-2.5 text-sm text-red-200"
          role="alert"
        >
          <Icon className="mt-0.5 size-4 shrink-0" name="alert" />
          {error}
        </div>
      ) : null}

      <Button className="mt-5 w-full" disabled={!value.trim() || busy} size="lg" type="submit">
        {busy ? 'Registering repository…' : 'Build local atlas'}
        <Icon className="size-4" name="arrow-right" />
      </Button>
    </form>
  )
}

function SourceOption({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean
  icon: 'folder' | 'repository'
  label: string
  onClick: () => void
}) {
  return (
    <button
      aria-pressed={active}
      className={`flex h-10 items-center justify-center gap-2 rounded-lg text-sm font-semibold transition-colors ${active ? 'bg-surface text-foreground shadow-sm' : 'text-muted hover:text-foreground'}`}
      onClick={onClick}
      type="button"
    >
      <Icon className="size-4" name={icon} />
      {label}
    </button>
  )
}
