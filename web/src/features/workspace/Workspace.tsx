import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api'
import { AnalysisFailure, AnalysisProgress } from '../analysis/AnalysisProgress'
import { useAnalysisEvents } from '../analysis/useAnalysisEvents'
import { GraphCanvas } from '../graph/GraphCanvas'
import { GraphError, RootRequired } from '../graph/GraphStates'
import { GraphToolbar } from '../graph/GraphToolbar'
import { EntityInspector } from '../source/EntityInspector'
import { useAtlasStore } from '../../store'
import type { ArchitectureCrumb, Entity, GraphResponse, GraphView, Project } from '../../types'
import { WorkspaceSidebar } from './WorkspaceSidebar'

export function Workspace({ project }: { project: Project }) {
  const queryClient = useQueryClient()
  const liveRun = useAnalysisEvents(project)
  const reanalyze = useMutation({
    mutationFn: () => api.analyze(project.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })

  if (liveRun?.status === 'failed') {
    return (
      <AnalysisFailure
        busy={reanalyze.isPending}
        onRetry={() => reanalyze.mutate()}
        run={liveRun}
      />
    )
  }
  if (project.status !== 'ready') {
    return <AnalysisProgress run={liveRun} />
  }

  return <ReadyWorkspace project={project} />
}

function ReadyWorkspace({ project }: { project: Project }) {
  const { selectedEntityId, graphView, selectEntity, setGraphView, resetSelection } =
    useAtlasStore()
  const [architecturePath, setArchitecturePath] = useState<ArchitectureCrumb[]>([])

  const scopeId = architecturePath.at(-1)?.id ?? ''
  const hasGraphRoot = graphView === 'architecture' || Boolean(selectedEntityId)
  const graphQuery = useQuery({
    queryKey: ['graph', project.id, graphView, scopeId, selectedEntityId],
    queryFn: () => loadGraph(project.id, graphView, selectedEntityId, scopeId),
    enabled: hasGraphRoot,
  })

  const selectedQuery = useQuery({
    queryKey: ['entity', project.id, selectedEntityId],
    queryFn: () => api.entity(project.id, selectedEntityId),
    enabled: Boolean(selectedEntityId),
  })
  const selectedEntity = useMemo(
    () => graphQuery.data?.nodes.find((node) => node.id === selectedEntityId) ?? selectedQuery.data,
    [graphQuery.data, selectedEntityId, selectedQuery.data],
  )

  const explore = (entity: Entity) => {
    if (entity.kind !== 'module' && entity.kind !== 'file') {
      return
    }
    setGraphView('architecture')
    selectEntity(entity.id)
    setArchitecturePath((current) => {
      const existingIndex = current.findIndex((crumb) => crumb.id === entity.id)
      if (existingIndex >= 0) {
        return current.slice(0, existingIndex + 1)
      }
      return [...current, { id: entity.id, name: entity.name, kind: entity.kind }]
    })
  }

  const exploreById = (entityId: string) => {
    const entity = graphQuery.data?.nodes.find((node) => node.id === entityId)
    if (entity) {
      explore(entity)
    }
  }

  const changeView = (view: GraphView) => {
    setGraphView(view)
  }

  return (
    <div className="workspace">
      <WorkspaceSidebar
        activeView={graphView}
        graph={graphQuery.data}
        projectId={project.id}
        onSelect={selectEntity}
        onViewChange={changeView}
      />

      <main className="graph-panel">
        <GraphToolbar
          architecturePath={architecturePath}
          graph={graphQuery.data}
          view={graphView}
          onNavigate={(index) => {
            setArchitecturePath(index < 0 ? [] : architecturePath.slice(0, index + 1))
            resetSelection()
          }}
        />

        {!hasGraphRoot ? (
          <RootRequired view={graphView} />
        ) : graphQuery.isError ? (
          <GraphError />
        ) : (
          <GraphCanvas
            graph={graphQuery.data}
            mode={graphView}
            onExplore={exploreById}
            onSelect={selectEntity}
            selectedEntityId={selectedEntityId}
          />
        )}
      </main>

      <EntityInspector
        entity={selectedEntity}
        onClose={resetSelection}
        onExplore={explore}
        onViewChange={changeView}
        projectId={project.id}
      />
    </div>
  )
}

function loadGraph(
  projectId: string,
  view: GraphView,
  selectedEntityId: string,
  scopeId: string,
): Promise<GraphResponse> {
  if (view === 'architecture') {
    return api.architecture(projectId, scopeId)
  }
  if (view === 'flow') {
    return api.flow(projectId, selectedEntityId)
  }
  return api.impact(projectId, selectedEntityId)
}
