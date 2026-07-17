import type { AnalysisRun } from '../../types'

const stages = ['discovering', 'parsing', 'resolving', 'persisting']

export function AnalysisProgress({ run }: { run?: AnalysisRun }) {
  const percent = progressPercent(run)

  return (
    <main className="analysis-screen">
      <section className="progress-card">
        <div className="progress-heading">
          <span className="status-pulse" aria-hidden="true" />
          <div>
            <strong>{run?.message || 'Preparing the local analyzer'}</strong>
            <span>{readableStage(run?.stage)}</span>
          </div>
          <b>{percent}%</b>
        </div>
        <div
          aria-label={`Analysis ${percent}% complete`}
          aria-valuemax={100}
          aria-valuemin={0}
          aria-valuenow={percent}
          className="progress-track"
          role="progressbar"
        >
          <i style={{ width: `${percent}%` }} />
        </div>
      </section>

      <div className="pipeline" aria-label="Analysis stages">
        {stages.map((stage, index) => (
          <div key={stage} className={stageState(stage, run?.stage)}>
            <span>{index + 1}</span>
            <small>{stage}</small>
          </div>
        ))}
      </div>
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
    <main className="analysis-screen">
      <section className="failure-card">
        <span>Analysis stopped</span>
        <h1>CodeAtlas kept your last valid graph.</h1>
        <p>{run.errorMessage || 'The repository could not be analyzed.'}</p>
        <button className="button primary" disabled={busy} onClick={onRetry}>
          {busy ? 'Queueing…' : 'Try analysis again'}
        </button>
      </section>
    </main>
  )
}

function progressPercent(run?: AnalysisRun): number {
  if (run?.stage === 'ready') {
    return 100
  }
  if (run?.total) {
    return Math.min(99, Math.round((run.completed / run.total) * 100))
  }
  return 8
}

function readableStage(stage?: string): string {
  return (stage || 'queued').replaceAll('_', ' ')
}

function stageState(stage: string, currentStage?: string): string {
  const currentIndex = stages.indexOf(currentStage ?? '')
  const stageIndex = stages.indexOf(stage)
  if (stageIndex < currentIndex || currentStage === 'ready') {
    return 'complete'
  }
  if (stageIndex === currentIndex) {
    return 'active'
  }
  return ''
}
