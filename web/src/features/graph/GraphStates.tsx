import { Icon } from '../../components/Icon'

export function GraphError() {
  return (
    <div className="grid h-full place-items-center px-6 text-center">
      <div className="max-w-sm">
        <span className="mx-auto grid size-14 place-items-center rounded-2xl border border-danger/25 bg-danger/8 text-danger-foreground">
          <Icon className="size-5" name="alert" />
        </span>
        <h3 className="mt-5 text-sm font-semibold text-foreground">
          The graph could not be loaded
        </h3>
        <p className="mt-2 text-xs leading-5 text-muted">
          The last valid analysis remains available. Try another root or refresh the project.
        </p>
      </div>
    </div>
  )
}
