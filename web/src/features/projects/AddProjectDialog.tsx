import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, APIError } from '../../api'
import { Dialog } from '../../components/Dialog'
import { Icon } from '../../components/Icon'
import { ProjectForm, type ProjectInput } from './ProjectForm'

export function AddProjectDialog({
  open,
  onClose,
  onCreated,
}: {
  open: boolean
  onClose: () => void
  onCreated: (id: string) => void
}) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: (source: ProjectInput) => api.createProject(source),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
      onCreated(created.projectId)
    },
  })

  const error = mutation.error
    ? mutation.error instanceof APIError
      ? mutation.error.message
      : 'Could not add this repository.'
    : undefined

  return (
    <Dialog
      description="Analyze a local folder or create an application-managed Git clone."
      onClose={onClose}
      open={open}
      title="Add repository"
    >
      <ProjectForm
        busy={mutation.isPending}
        error={error}
        onSubmit={(source) => mutation.mutate(source)}
      />
      <div className="mx-5 mb-5 flex gap-3 rounded-xl border border-primary/20 bg-primary/5 p-4 sm:mx-6 sm:mb-6">
        <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
          <Icon className="size-4" name="shield" />
        </span>
        <div>
          <strong className="text-sm font-semibold text-foreground">
            Source stays on this machine
          </strong>
          <p className="mt-1 text-xs leading-5 text-muted">
            Only derived graph metadata and file hashes are stored in PostgreSQL.
          </p>
        </div>
      </div>
    </Dialog>
  )
}
