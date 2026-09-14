import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, APIError } from './api'
import { Brand } from './components/Brand'
import { Button } from './components/Button'
import { ConfirmDialog } from './components/ConfirmDialog'
import { Icon } from './components/Icon'
import { AnalysisProgress } from './features/analysis/AnalysisProgress'
import { AddProjectDialog } from './features/projects/AddProjectDialog'
import { Onboarding } from './features/projects/Onboarding'
import { ProjectHeader } from './features/projects/ProjectHeader'
import { Workspace } from './features/workspace/Workspace'
import { useAtlasStore } from './store'
import type { Project } from './types'

const emptyProjects: Project[] = []

export default function App() {
  const queryClient = useQueryClient()
  const projectId = useAtlasStore((state) => state.projectId)
  const setProject = useAtlasStore((state) => state.setProject)
  const [addProjectOpen, setAddProjectOpen] = useState(false)
  const [projectToRemove, setProjectToRemove] = useState<Project>()
  const [shutdownOpen, setShutdownOpen] = useState(false)
  const [stopped, setStopped] = useState(false)

  const projectsQuery = useQuery({
    queryKey: ['projects'],
    queryFn: ({ signal }) => api.projects(signal),
    enabled: !stopped,
    refetchInterval: (query) => (!stopped && query.state.data ? 3_000 : false),
  })
  const projects = projectsQuery.data?.projects ?? emptyProjects
  const project = projects.find((item) => item.id === projectId)

  useEffect(() => {
    if (!projectsQuery.isFetched) return
    if (!projectId || !projects.some((item) => item.id === projectId)) {
      setProject(projects[0]?.id ?? '')
    }
  }, [projectId, projects, projectsQuery.isFetched, setProject])

  const reanalyze = useMutation({
    mutationFn: (id: string) => api.analyze(id),
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: ['projects'] }),
  })

  const removeProject = useMutation({
    mutationFn: (id: string) => api.deleteProject(id),
    onSuccess: async (_, removedId) => {
      setProject(projects.find((item) => item.id !== removedId)?.id ?? '')
      setProjectToRemove(undefined)
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })

  const shutdown = useMutation({
    mutationFn: api.shutdown,
    onSuccess: () => {
      setShutdownOpen(false)
      setStopped(true)
    },
  })

  if (stopped) return <StoppedScreen />

  return (
    <div className="h-dvh overflow-hidden">
      <ProjectHeader
        onAdd={() => setAddProjectOpen(true)}
        onChange={setProject}
        onReanalyze={(id) => reanalyze.mutate(id)}
        onRemove={(nextProject) => {
          removeProject.reset()
          setProjectToRemove(nextProject)
        }}
        onShutdown={() => {
          shutdown.reset()
          setShutdownOpen(true)
        }}
        projects={projects}
        reanalyzing={reanalyze.isPending}
        selectedProject={project}
      />

      <AppContent
        error={projectsQuery.isError && !projectsQuery.data}
        loading={projectsQuery.isLoading}
        onAdd={() => setAddProjectOpen(true)}
        onRetry={() => void projectsQuery.refetch()}
        project={project}
      />

      <AddProjectDialog
        onClose={() => setAddProjectOpen(false)}
        onCreated={(id) => {
          setProject(id)
          setAddProjectOpen(false)
        }}
        open={addProjectOpen}
      />

      <ConfirmDialog
        busy={removeProject.isPending}
        confirmLabel="Remove repository"
        danger
        error={mutationMessage(removeProject.error)}
        onCancel={() => {
          removeProject.reset()
          setProjectToRemove(undefined)
        }}
        onConfirm={() => projectToRemove && removeProject.mutate(projectToRemove.id)}
        open={Boolean(projectToRemove)}
        title="Remove repository?"
      >
        <p>
          CodeAtlas will delete the derived graph for{' '}
          <strong className="text-foreground">{projectToRemove?.name}</strong>. Local source files
          will not be touched.
        </p>
        {projectToRemove?.managedClone ? (
          <p className="mt-3">The application-managed clone will also be removed.</p>
        ) : null}
      </ConfirmDialog>

      <ConfirmDialog
        busy={shutdown.isPending}
        confirmLabel="Stop local server"
        danger
        error={mutationMessage(shutdown.error)}
        onCancel={() => {
          shutdown.reset()
          setShutdownOpen(false)
        }}
        onConfirm={() => shutdown.mutate()}
        open={shutdownOpen}
        title="Stop CodeAtlas?"
      >
        <p>
          This stops the local Go process. Project metadata remains in the local database for the
          next launch.
        </p>
      </ConfirmDialog>
    </div>
  )
}

function AppContent({
  error,
  loading,
  project,
  onAdd,
  onRetry,
}: {
  error: boolean
  loading: boolean
  project?: Project
  onAdd: () => void
  onRetry: () => void
}) {
  if (loading) return <AnalysisProgress />
  if (error) return <ProjectLoadError onRetry={onRetry} />
  if (!project) return <Onboarding onAdd={onAdd} />
  return <Workspace key={project.id} project={project} />
}

function ProjectLoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <main className="grid min-h-[calc(100dvh-4rem)] place-items-center px-5 py-10">
      <section className="w-full max-w-lg rounded-panel border border-danger/30 bg-panel p-8 text-center shadow-[0_30px_100px_var(--ui-shadow)]">
        <span className="mx-auto grid size-12 place-items-center rounded-xl bg-danger/10 text-danger-foreground">
          <Icon className="size-5" name="alert" />
        </span>
        <h1 className="mt-5 text-lg font-semibold tracking-tight">Could not reach the local API</h1>
        <p className="mt-2 text-sm leading-6 text-muted">
          The interface is loaded, but the Go server did not return project metadata.
        </p>
        <Button className="mt-6" onClick={onRetry}>
          <Icon className="size-4" name="refresh" />
          Try again
        </Button>
      </section>
    </main>
  )
}

function mutationMessage(error: Error | null): string | undefined {
  if (!error) return undefined
  return error instanceof APIError ? error.message : 'The local request could not be completed.'
}

function StoppedScreen() {
  return (
    <main className="grid min-h-dvh place-items-center px-6">
      <section className="w-full max-w-lg rounded-panel border border-border-strong bg-panel p-8 text-center shadow-[0_30px_100px_var(--ui-shadow)]">
        <div className="mb-8 flex justify-center">
          <Brand />
        </div>
        <span className="mx-auto grid size-14 place-items-center rounded-2xl border border-primary/25 bg-primary/10 text-primary">
          <Icon className="size-6" name="power" />
        </span>
        <h1 className="mt-6 text-2xl font-semibold tracking-tight">CodeAtlas is stopped</h1>
        <p className="mt-3 text-sm leading-6 text-muted">
          You can close this tab. Start the local server again when you want to continue.
        </p>
        <code className="mt-6 inline-flex rounded-lg border border-border bg-canvas-raised px-4 py-2 font-mono text-sm text-primary">
          codeatlas serve
        </code>
      </section>
    </main>
  )
}
