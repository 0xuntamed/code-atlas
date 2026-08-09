import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api'
import { filterGraph } from '../../lib/graph'
import { useAtlasStore } from '../../store'
import type { ArchitectureCrumb, Entity, GraphResponse, GraphView, Project } from '../../types'
import { AnalysisFailure, AnalysisProgress } from '../analysis/AnalysisProgress'
import { useAnalysisEvents } from '../analysis/useAnalysisEvents'
import { GraphCanvas } from '../graph/GraphCanvas'
import { GraphError, RootRequired, TraceNeedsSymbol } from '../graph/GraphStates'
import { GraphToolbar } from '../graph/GraphToolbar'
import { EntityInspector } from '../source/EntityInspector'
import { WorkspaceSidebar } from './WorkspaceSidebar'

export function Workspace({ project }: { project: Project }) {
  const queryClient = useQueryClient()
  const liveRun = useAnalysisEvents(project)
  const reanalyze = useMutation({
    mutationFn: () => api.analyze(project.id),
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: ['projects'] }),
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
  if (project.status !== 'ready') return <AnalysisProgress run={liveRun} />
  return <ReadyWorkspace project={project} />
}

function ReadyWorkspace({ project }: { project: Project }) {
  const selectedEntityId = useAtlasStore((state) => state.selectedEntityId)
  const graphView = useAtlasStore((state) => state.graphView)
  const showTests = useAtlasStore((state) => state.showTests)
  const showReferences = useAtlasStore((state) => state.showReferences)
  const selectEntity = useAtlasStore((state) => state.selectEntity)
  const setGraphView = useAtlasStore((state) => state.setGraphView)
  const toggleTests = useAtlasStore((state) => state.toggleTests)
  const toggleReferences = useAtlasStore((state) => state.toggleReferences)
  const resetSelection = useAtlasStore((state) => state.resetSelection)
  const [architecturePath, setArchitecturePath] = useState<ArchitectureCrumb[]>([])

  const scopeId = architecturePath.at(-1)?.id ?? ''
  const hasGraphRoot = graphView === 'architecture' || Boolean(selectedEntityId)
  const graphRootId = graphView === 'architecture' ? scopeId : selectedEntityId
  const graphQuery = useQuery({
    queryKey: ['graph', project.id, project.activeRunId, graphView, graphRootId],
    queryFn: ({ signal }) => loadGraph(project.id, graphView, selectedEntityId, scopeId, signal),
    enabled: hasGraphRoot,
  })

  const entityInGraph = graphQuery.data?.nodes.find((node) => node.id === selectedEntityId)
  const selectedQuery = useQuery({
    queryKey: ['entity', project.id, selectedEntityId],
    queryFn: ({ signal }) => api.entity(project.id, selectedEntityId, signal),
    enabled: Boolean(selectedEntityId) && !entityInGraph,
  })
  const selectedEntity = entityInGraph ?? selectedQuery.data

  // Flow and impact traversals exist to surface what a symbol reaches, including
  // external and unresolved calls. Hiding reference nodes there empties the view,
  // so those modes always keep them regardless of the architecture-only filter.
  const traceView = graphView !== 'architecture'
  const filtered = useMemo(
    () =>
      filterGraph(
        graphQuery.data,
        { showTests, showReferences: showReferences || traceView },
        selectedEntityId,
      ),
    [graphQuery.data, selectedEntityId, showReferences, showTests, traceView],
  )

  // Flow/impact start from a function, method, or route — never a module or file.
  const traceRootIsContainer =
    traceView && (selectedEntity?.kind === 'module' || selectedEntity?.kind === 'file')

  const explore = (entity: Entity) => {
    if (entity.kind !== 'module' && entity.kind !== 'file') return
    setGraphView('architecture')
    selectEntity(entity.id)
    setArchitecturePath((current) => {
      const existingIndex = current.findIndex((crumb) => crumb.id === entity.id)
      if (existingIndex >= 0) return current.slice(0, existingIndex + 1)
      return [...current, { id: entity.id, name: entity.name, kind: entity.kind }]
    })
  }

  const exploreById = (entityId: string) => {
    const entity = graphQuery.data?.nodes.find((node) => node.id === entityId)
    if (entity) explore(entity)
  }

  const navigateArchitecture = (index: number) => {
    if (index < 0) {
      setArchitecturePath([])
      resetSelection()
      return
    }
    const crumb = architecturePath[index]
    setArchitecturePath((current) => current.slice(0, index + 1))
    selectEntity(crumb.id)
  }

  return (
    <div className="relative grid h-[calc(100dvh-4rem)] grid-rows-[auto_minmax(0,1fr)] overflow-hidden md:grid-cols-[17rem_minmax(0,1fr)] md:grid-rows-1 xl:grid-cols-[17rem_minmax(0,1fr)_22.5rem]">
      <WorkspaceSidebar
        activeView={graphView}
        filtered={filtered}
        onSelect={selectEntity}
        onToggleReferences={toggleReferences}
        onToggleTests={toggleTests}
        onViewChange={setGraphView}
        projectId={project.id}
        showReferences={showReferences}
        showTests={showTests}
      />

      <main className="grid min-h-0 min-w-0 grid-rows-[auto_minmax(0,1fr)] bg-canvas/35">
        <GraphToolbar
          architecturePath={architecturePath}
          filtered={filtered}
          onNavigate={navigateArchitecture}
          view={graphView}
        />
        <section className="relative min-h-0">
          {!hasGraphRoot ? (
            <RootRequired view={graphView} />
          ) : traceRootIsContainer ? (
            <TraceNeedsSymbol view={graphView} />
          ) : graphQuery.isError ? (
            <GraphError />
          ) : (
            <GraphCanvas
              graph={filtered.graph}
              hiddenCount={filtered.hiddenTotal}
              mode={graphView}
              onExplore={exploreById}
              onSelect={selectEntity}
              selectedEntityId={selectedEntityId}
            />
          )}
        </section>
      </main>

      <EntityInspector
        entity={selectedEntity}
        key={selectedEntity?.id ?? 'empty'}
        onClose={resetSelection}
        onExplore={explore}
        onViewChange={setGraphView}
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
  signal: AbortSignal,
): Promise<GraphResponse> {
  if (view === 'architecture') return api.architecture(projectId, scopeId, signal)
  if (view === 'flow') return api.flow(projectId, selectedEntityId, signal)
  return api.impact(projectId, selectedEntityId, signal)
}
