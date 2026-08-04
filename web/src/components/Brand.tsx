import { Icon } from './Icon'

export function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <div aria-label="CodeAtlas" className="flex items-center gap-3">
      <span className="relative grid size-9 shrink-0 place-items-center overflow-hidden rounded-xl border border-primary/30 bg-primary/10 text-primary shadow-[inset_0_1px_0_oklch(1_0_0/0.08)]">
        <span className="absolute inset-0 bg-[radial-gradient(circle_at_30%_20%,oklch(0.82_0.15_173/0.24),transparent_65%)]" />
        <Icon className="relative size-[19px]" name="layers" />
      </span>
      {!compact ? (
        <span className="flex flex-col leading-none">
          <strong className="text-[15px] font-bold tracking-[-0.025em] text-foreground">
            CodeAtlas
          </strong>
          <small className="mt-1 text-[9px] font-semibold uppercase tracking-[0.22em] text-dim">
            local intelligence
          </small>
        </span>
      ) : null}
    </div>
  )
}
