import { Brand } from '../../components/Brand'
import { Button } from '../../components/Button'
import { Icon, type IconName } from '../../components/Icon'

const capabilities: Array<{ icon: IconName; title: string; copy: string }> = [
  { icon: 'architecture', title: 'Architecture', copy: 'Progressive modules, files, and symbols.' },
  { icon: 'flow', title: 'Execution flow', copy: 'Trace routes, middleware, and calls.' },
  { icon: 'impact', title: 'Change impact', copy: 'See upstream and downstream dependents.' },
]

export function Onboarding({ onAdd }: { onAdd: () => void }) {
  return (
    <main className="relative min-h-[calc(100dvh-4rem)] overflow-y-auto px-5 py-10 sm:px-8 lg:grid lg:place-items-center lg:py-14">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-35 [background-image:linear-gradient(var(--ui-line)_1px,transparent_1px),linear-gradient(90deg,var(--ui-line)_1px,transparent_1px)] [background-size:44px_44px] [mask-image:radial-gradient(circle_at_center,black,transparent_75%)]"
      />
      <section className="relative mx-auto w-full max-w-6xl overflow-hidden rounded-[1.4rem] border border-border-strong bg-panel/90 shadow-[0_40px_120px_var(--ui-shadow)] backdrop-blur">
        <div className="grid lg:grid-cols-[1.08fr_0.92fr]">
          <div className="p-7 sm:p-10 lg:p-14">
            <Brand />
            <span className="mt-14 inline-flex items-center gap-2 rounded-full border border-primary/25 bg-primary/8 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[0.18em] text-primary">
              <Icon className="size-3.5" name="lock" />
              Private by architecture
            </span>
            <h1 className="mt-6 max-w-2xl text-4xl font-semibold leading-[1.06] tracking-[-0.04em] text-foreground sm:text-5xl">
              See the system inside your codebase.
            </h1>
            <p className="mt-5 max-w-xl text-base leading-7 text-muted">
              Explore architecture, runtime flow, and blast radius across JavaScript, TypeScript,
              Go, and Python—without uploading a line of source.
            </p>
            <Button className="mt-8" onClick={onAdd} size="lg">
              Add your first repository
              <Icon className="size-4" name="arrow-right" />
            </Button>
            <div className="mt-8 flex flex-wrap gap-x-5 gap-y-2 text-xs font-medium text-dim">
              {['No source uploads', 'No AI calls', 'Loopback only'].map((item) => (
                <span className="inline-flex items-center gap-1.5" key={item}>
                  <Icon className="size-3.5 text-primary" name="check" />
                  {item}
                </span>
              ))}
            </div>
          </div>

          <div className="border-t border-border bg-canvas-raised/75 p-6 sm:p-8 lg:border-l lg:border-t-0 lg:p-10">
            <div className="rounded-panel border border-border bg-panel p-4 shadow-xl">
              <div className="flex items-center justify-between border-b border-border pb-4">
                <div className="flex gap-1.5">
                  <i className="size-2 rounded-full bg-danger/70" />
                  <i className="size-2 rounded-full bg-warning/70" />
                  <i className="size-2 rounded-full bg-primary/70" />
                </div>
                <span className="font-mono text-[9px] uppercase tracking-[0.16em] text-dim">
                  derived graph · local
                </span>
              </div>
              <div className="space-y-3 py-5">
                {capabilities.map((item, index) => (
                  <article
                    className="flex items-start gap-4 rounded-xl border border-border bg-canvas-raised p-4"
                    key={item.title}
                  >
                    <span className="grid size-10 shrink-0 place-items-center rounded-lg border border-primary/20 bg-primary/8 text-primary">
                      <Icon className="size-[18px]" name={item.icon} />
                    </span>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-[9px] text-dim">0{index + 1}</span>
                        <h2 className="text-sm font-semibold text-foreground">{item.title}</h2>
                      </div>
                      <p className="mt-1 text-xs leading-5 text-muted">{item.copy}</p>
                    </div>
                  </article>
                ))}
              </div>
              <div className="flex items-center gap-2 border-t border-border pt-4 text-[10px] font-semibold uppercase tracking-[0.16em] text-dim">
                <span className="size-1.5 animate-pulse-soft rounded-full bg-primary" />
                PostgreSQL stores metadata, never source
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  )
}
