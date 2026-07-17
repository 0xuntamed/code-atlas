import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, APIError } from '../../api'
import { Modal } from '../../components/Modal'
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
    <Modal
      description="Analyze another local folder or locally clone a Git repository."
      onClose={onClose}
      open={open}
      title="Add repository"
    >
      <ProjectForm
        busy={mutation.isPending}
        compact
        error={error}
        onSubmit={(source) => mutation.mutate(source)}
      />
      <PrivacyNote />
    </Modal>
  )
}

export function PrivacyNote() {
  return (
    <div className="privacy-note">
      <span className="privacy-shield" aria-hidden="true">
        ✓
      </span>
      <div>
        <strong>Source stays on this machine</strong>
        <p>Only derived graph metadata and file hashes are stored in PostgreSQL.</p>
      </div>
    </div>
  )
}
