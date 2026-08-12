import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api'
import { isTraceable } from '../../lib/entityKinds'
import { useAtlasStore } from '../../store'
import type { Entity, Project } from '../../types'
import { AnalysisFailure, AnalysisProgress } from '../analysis/AnalysisProgress'
import { useAnalysisEvents } from '../analysis/useAnalysisEvents'
import { GraphCanvas } from '../graph/GraphCanvas'
import { GraphError } from '../graph/GraphStates'
import { GraphToolbar } from '../graph/GraphToolbar'
import { EntityInspector } from '../source/EntityInspector'
import { useWorkspaceGraph } from './useWorkspaceGraph'
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
  const lens = useAtlasStore((state) => state.lens)
  const scopePath = useAtlasStore((state) => state.scopePath)
  const showTests = useAtlasStore((state) => state.showTests)
  const showReferences = useAtlasStore((state) => state.showReferences)
  const selectEntity = useAtlasStore((state) => state.selectEntity)
  const setLens = useAtlasStore((state) => state.setLens)
  const openScope = useAtlasStore((state) => state.openScope)
  const navigateScope = useAtlasStore((state) => state.navigateScope)
  const toggleTests = useAtlasStore((state) => state.toggleTests)
  const toggleReferences = useAtlasStore((state) => state.toggleReferences)
  const resetSelection = useAtlasStore((state) => state.resetSelection)

  const graph = useWorkspaceGraph(project)
  const lensEnabled = isTraceable(graph.selectedEntity?.kind)

  // Drilling in only applies to containers; the canvas guards double-click too.
  const explore = (entity: Entity) => {
    if (entity.kind === 'module' || entity.kind === 'file') {
      openScope({ id: entity.id, name: entity.name, kind: entity.kind })
    }
  }
  const exploreById = (entityId: string) => {
    const entity = graph.displayGraph?.nodes.find((node) => node.id === entityId)
    if (entity) explore(entity)
  }

  return (
    <div className="relative grid h-[calc(100dvh-4rem)] grid-rows-[auto_minmax(0,1fr)] overflow-hidden md:grid-cols-[17rem_minmax(0,1fr)] md:grid-rows-1 xl:grid-cols-[17rem_minmax(0,1fr)_22.5rem]">
      <WorkspaceSidebar
        activeLens={lens}
        graph={graph.displayGraph}
        hiddenTotal={graph.hiddenTotal}
        lensEnabled={lensEnabled}
        onLensChange={setLens}
        onSelect={selectEntity}
        onToggleReferences={toggleReferences}
        onToggleTests={toggleTests}
        projectId={project.id}
        showReferences={showReferences}
        showTests={showTests}
      />

      <main className="grid min-h-0 min-w-0 grid-rows-[auto_minmax(0,1fr)] bg-canvas/35">
        <GraphToolbar
          graph={graph.displayGraph}
          hiddenTotal={graph.hiddenTotal}
          lens={lens}
          lensActive={graph.lensActive}
          onNavigate={navigateScope}
          scopePath={scopePath}
        />
        <section className="relative min-h-0">
          {graph.isError ? (
            <GraphError />
          ) : (
            <GraphCanvas
              graph={graph.displayGraph}
              highlightEdgeIds={graph.highlightEdgeIds}
              highlightNodeIds={graph.highlightNodeIds}
              hiddenCount={graph.hiddenTotal}
              layoutMode={graph.lensActive ? lens : 'structure'}
              lensActive={graph.lensActive}
              onExplore={exploreById}
              onSelect={selectEntity}
              selectedEntityId={selectedEntityId}
            />
          )}
        </section>
      </main>

      <EntityInspector
        entity={graph.selectedEntity}
        key={graph.selectedEntity?.id ?? 'empty'}
        onClose={resetSelection}
        onExplore={explore}
        onSetLens={setLens}
        projectId={project.id}
      />
    </div>
  )
}
