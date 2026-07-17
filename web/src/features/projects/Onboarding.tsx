import { Logo } from '../../components/Logo'

export function Onboarding({ onAdd }: { onAdd: () => void }) {
  return (
    <main className="onboarding">
      <div className="ambient-grid" aria-hidden="true" />
      <section className="welcome-card">
        <Logo />
        <span className="eyebrow">Private by architecture</span>
        <h1>Understand the system inside your codebase.</h1>
        <p>
          Trace routes, execution flow, and change impact across JavaScript, TypeScript, Go, and
          Python—without uploading source code.
        </p>
        <button className="button primary welcome-action" onClick={onAdd}>
          Add your first repository
          <span aria-hidden="true">→</span>
        </button>
        <div className="privacy-points">
          <span>No source uploads</span>
          <span>No AI calls</span>
          <span>Loopback only</span>
        </div>
      </section>
    </main>
  )
}
