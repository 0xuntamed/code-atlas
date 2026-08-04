import { Button } from '../../components/Button'
import { Icon } from '../../components/Icon'
import type { AnalysisRun } from '../../types'

const stages = ['acquiring', 'discovering', 'parsing', 'resolving', 'persisting']

export function AnalysisProgress({ run }: { run?: AnalysisRun }) {
  const percent = progressPercent(run)
  return (
    <main className="grid min-h-[calc(100dvh-4rem)] place-items-center overflow-y-auto px-5 py-10">
      <section className="w-full max-w-2xl rounded-panel border border-border-strong bg-panel p-6 shadow-[0_30px_100px_var(--ui-shadow)] sm:p-8">
        <div className="flex items-start gap-4">
          <span className="relative mt-0.5 grid size-11 shrink-0 place-items-center rounded-xl border border-primary/25 bg-primary/10 text-primary">
            <span className="absolute inset-0 animate-pulse-soft rounded-xl bg-primary/5" />
            <Icon className="relative size-5" name="spark" />
          </span>
          <div>
            <span className="text-[10px] font-bold uppercase tracking-[0.18em] text-primary">
              {readableStage(run?.stage)}
            </span>
            <h1 className="mt-1 text-lg font-semibold tracking-tight text-foreground">
              {run?.message || 'Preparing the local analyzer'}
            </h1>
          </div>
          <b className="ml-auto font-mono text-sm text-muted">{percent}%</b>
        </div>

        <div
          aria-label={`Analysis ${percent}% complete`}
          aria-valuemax={100}
          aria-valuemin={0}
          aria-valuenow={percent}
          className="mt-6 h-1.5 overflow-hidden rounded-full bg-canvas-raised"
          role="progressbar"
        >
          <i
            className="block h-full rounded-full bg-gradient-to-r from-primary-strong to-primary transition-[width] duration-500"
            style={{ width: `${percent}%` }}
          />
        </div>

        <div className="mt-7 grid grid-cols-2 gap-2 sm:grid-cols-5" aria-label="Analysis stages">
          {stages.map((stage, index) => {
            const state = stageState(stage, run?.stage)
            return (
              <div
                className={`rounded-lg border px-3 py-3 ${state === 'complete' ? 'border-primary/20 bg-primary/5 text-primary' : state === 'active' ? 'border-accent/30 bg-accent/8 text-accent-foreground' : 'border-border bg-canvas-raised/55 text-dim'}`}
                key={stage}
              >
                <span className="font-mono text-[9px]">0{index + 1}</span>
                <small className="mt-1 block text-[10px] font-bold uppercase tracking-[0.11em]">
                  {stage}
                </small>
              </div>
            )
          })}
        </div>
        <p className="mt-6 flex items-center gap-2 text-xs leading-5 text-dim">
          <Icon className="size-3.5 text-primary" name="shield" />
          Source buffers are discarded after parsing. Only derived metadata is persisted.
        </p>
      </section>
    </main>
  )
}

export function AnalysisFailure({
  run,
  busy,
  onRetry,
}: {
  run: AnalysisRun
  busy: boolean
  onRetry: () => void
}) {
  return (
    <main className="grid min-h-[calc(100dvh-4rem)] place-items-center px-5 py-10">
      <section className="w-full max-w-xl rounded-panel border border-danger/30 bg-panel p-7 shadow-[0_30px_100px_var(--ui-shadow)] sm:p-9">
        <span className="grid size-11 place-items-center rounded-xl bg-danger/10 text-danger-foreground">
          <Icon className="size-5" name="alert" />
        </span>
        <span className="mt-6 block text-[10px] font-bold uppercase tracking-[0.18em] text-danger-foreground">
          Analysis stopped
        </span>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">
          CodeAtlas kept your last valid graph.
        </h1>
        <p className="mt-3 text-sm leading-6 text-muted">
          {run.errorMessage || 'The repository could not be analyzed.'}
        </p>
        <Button className="mt-6" disabled={busy} onClick={onRetry}>
          <Icon className="size-4" name="refresh" />
          {busy ? 'Queueing…' : 'Try analysis again'}
        </Button>
      </section>
    </main>
  )
}

function progressPercent(run?: AnalysisRun): number {
  if (run?.stage === 'ready') return 100
  if (run?.total) return Math.min(99, Math.round((run.completed / run.total) * 100))
  return 8
}

function readableStage(stage?: string): string {
  return (stage || 'queued').replaceAll('_', ' ')
}

function stageState(stage: string, currentStage?: string): 'complete' | 'active' | 'pending' {
  const currentIndex = stages.indexOf(currentStage ?? '')
  const stageIndex = stages.indexOf(stage)
  if (stageIndex < currentIndex || currentStage === 'ready') return 'complete'
  if (stageIndex === currentIndex) return 'active'
  return 'pending'
}
