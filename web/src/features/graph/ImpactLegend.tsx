const breaks = { color: '#fb7185', label: 'Breaks (dependents)' }
const relies = { color: '#38bdf8', label: 'Relies on (dependencies)' }
const both = { color: '#a78bfa', label: 'Both' }
const selecting = { color: 'var(--color-primary, #6ea8fe)', label: 'Changing this' }
const changed = { color: '#fbbf24', label: 'Changed' }

// Color key for the blast-radius view. In review mode the seed swatch reads
// "Changed" (amber); otherwise it reads "Changing this" (the selected node).
export function ImpactLegend({
  className = '',
  review = false,
}: {
  className?: string
  review?: boolean
}) {
  const swatches = [review ? changed : selecting, breaks, relies, both]
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
