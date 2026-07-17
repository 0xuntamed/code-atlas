import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import { ConfirmDialog } from './components/ConfirmDialog'
import { Logo } from './components/Logo'
import { AnalysisProgress } from './features/analysis/AnalysisProgress'
import { AddProjectDialog } from './features/projects/AddProjectDialog'
import { Onboarding } from './features/projects/Onboarding'
import { ProjectHeader } from './features/projects/ProjectHeader'
import { Workspace } from './features/workspace/Workspace'
import { useAtlasStore } from './store'
import type { Project } from './types'

export default function App() {
  const queryClient = useQueryClient()
  const { projectId, setProject } = useAtlasStore()
  const [addProjectOpen, setAddProjectOpen] = useState(false)
  const [projectToRemove, setProjectToRemove] = useState<Project>()
  const [shutdownOpen, setShutdownOpen] = useState(false)
  const [stopped, setStopped] = useState(false)

  const projectsQuery = useQuery({
    queryKey: ['projects'],
    queryFn: api.projects,
    refetchInterval: 2_500,
  })
  const projects = useMemo(() => projectsQuery.data?.projects ?? [], [projectsQuery.data?.projects])
  const project = projects.find((item) => item.id === projectId)

  useEffect(() => {
    if (!projectId && projects[0]) {
      setProject(projects[0].id)
    }
  }, [projectId, projects, setProject])

  const reanalyze = useMutation({
    mutationFn: (id: string) => api.analyze(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })

  const removeProject = useMutation({
    mutationFn: (id: string) => api.deleteProject(id),
    onSuccess: async (_, removedId) => {
      const nextProject = projects.find((item) => item.id !== removedId)
      setProject(nextProject?.id ?? '')
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

  if (stopped) {
    return <StoppedScreen />
  }

  return (
    <div className="app-shell">
      <ProjectHeader
        projects={projects}
        selectedProject={project}
        onAdd={() => setAddProjectOpen(true)}
        onChange={setProject}
        onReanalyze={(id) => reanalyze.mutate(id)}
        onRemove={setProjectToRemove}
        onShutdown={() => setShutdownOpen(true)}
        reanalyzing={reanalyze.isPending}
      />

      <AppContent
        loading={projectsQuery.isLoading}
        project={project}
        onAdd={() => setAddProjectOpen(true)}
      />

      <AddProjectDialog
        open={addProjectOpen}
        onClose={() => setAddProjectOpen(false)}
        onCreated={(id) => {
          setProject(id)
          setAddProjectOpen(false)
        }}
      />

      <ConfirmDialog
        open={Boolean(projectToRemove)}
        title="Remove repository?"
        confirmLabel="Remove repository"
        danger
        busy={removeProject.isPending}
        onCancel={() => setProjectToRemove(undefined)}
        onConfirm={() => projectToRemove && removeProject.mutate(projectToRemove.id)}
      >
        <p>
          CodeAtlas will delete the derived graph for <strong>{projectToRemove?.name}</strong>.
          Local source files will not be touched.
        </p>
        {projectToRemove?.managedClone && (
          <p>The application-managed clone will also be removed.</p>
        )}
      </ConfirmDialog>

      <ConfirmDialog
        open={shutdownOpen}
        title="Stop CodeAtlas?"
        confirmLabel="Stop local server"
        danger
        busy={shutdown.isPending}
        onCancel={() => setShutdownOpen(false)}
        onConfirm={() => shutdown.mutate()}
      >
        <p>
          This stops the local Go process. Your project metadata remains in PostgreSQL and will be
          available the next time you start CodeAtlas.
        </p>
      </ConfirmDialog>
    </div>
  )
}

function AppContent({
  loading,
  project,
  onAdd,
}: {
  loading: boolean
  project?: Project
  onAdd: () => void
}) {
  if (loading) {
    return <AnalysisProgress />
  }
  if (!project) {
    return <Onboarding onAdd={onAdd} />
  }
  return <Workspace key={project.id} project={project} />
}

function StoppedScreen() {
  return (
    <main className="stopped-screen">
      <Logo />
      <div className="stopped-mark" aria-hidden="true" />
      <h1>CodeAtlas is stopped.</h1>
      <p>You can close this tab. Start the local server again when you want to continue.</p>
      <code>codeatlas serve</code>
    </main>
  )
}
