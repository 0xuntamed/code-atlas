const swatches: { color: string; label: string }[] = [
  { color: 'var(--color-primary, #6ea8fe)', label: 'Changing this' },
  { color: '#fb7185', label: 'Breaks (dependents)' },
  { color: '#38bdf8', label: 'Relies on (dependencies)' },
  { color: '#a78bfa', label: 'Both' },
]

// Color key for the blast-radius view. Rendered only while a node is selected.
export function ImpactLegend({ className = '' }: { className?: string }) {
  return (
    <ul className={`flex flex-wrap items-center gap-x-3 gap-y-1 ${className}`}>
      {swatches.map((swatch) => (
        <li
          className="flex items-center gap-1.5 text-[9px] font-medium text-muted"
          key={swatch.label}
        >
          <span
            aria-hidden
            className="size-2.5 rounded-full ring-1 ring-inset ring-black/10"
            style={{ backgroundColor: swatch.color }}
          />
          {swatch.label}
        </li>
      ))}
    </ul>
  )
}
