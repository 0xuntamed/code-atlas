import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api'
import { Icon } from '../../components/Icon'
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
  const impactFilter = useAtlasStore((state) => state.impactFilter)
  const scopePath = useAtlasStore((state) => state.scopePath)
  const showTests = useAtlasStore((state) => state.showTests)
  const showReferences = useAtlasStore((state) => state.showReferences)
  const selectEntity = useAtlasStore((state) => state.selectEntity)
  const setImpactFilter = useAtlasStore((state) => state.setImpactFilter)
  const openScope = useAtlasStore((state) => state.openScope)
  const navigateScope = useAtlasStore((state) => state.navigateScope)
  const toggleTests = useAtlasStore((state) => state.toggleTests)
  const toggleReferences = useAtlasStore((state) => state.toggleReferences)
  const resetSelection = useAtlasStore((state) => state.resetSelection)
  const reviewMode = useAtlasStore((state) => state.reviewMode)
  const setReviewMode = useAtlasStore((state) => state.setReviewMode)

  // File-group expansion is ephemeral view state; a new selection resets it,
  // tracking the previous selection in state per React's "reset on change" pattern.
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set())
  const [lastSelection, setLastSelection] = useState(selectedEntityId)
  if (lastSelection !== selectedEntityId) {
    setLastSelection(selectedEntityId)
    setExpandedFiles(new Set())
  }
  const toggleGroup = (key: string) =>
    setExpandedFiles((current) => {
      const next = new Set(current)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })

  const graph = useWorkspaceGraph(project, expandedFiles)

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
        expandedCount={expandedFiles.size}
        graph={graph.displayGraph}
        hiddenTotal={graph.hiddenTotal}
        impactActive={graph.impactActive}
        impactFilter={impactFilter}
        onCollapseFiles={() => setExpandedFiles(new Set())}
        onImpactFilterChange={setImpactFilter}
        onSelect={selectEntity}
        onToggleReferences={toggleReferences}
        onToggleReview={() => setReviewMode(!reviewMode)}
        onToggleTests={toggleTests}
        projectId={project.id}
        reviewMode={reviewMode}
        reviewSummary={graph.reviewSummary}
        showReferences={showReferences}
        showTests={showTests}
      />

      <main className="grid min-h-0 min-w-0 grid-rows-[auto_minmax(0,1fr)] bg-canvas/35">
        <GraphToolbar
          graph={graph.displayGraph}
          hiddenTotal={graph.hiddenTotal}
          impactActive={graph.impactActive}
          onNavigate={navigateScope}
          reviewMode={reviewMode}
          scopePath={scopePath}
          selectedName={graph.selectedEntity?.name}
        />
        <section className="relative min-h-0">
          {graph.isError ? (
            <GraphError />
          ) : reviewMode && graph.reviewEmpty ? (
            <ReviewEmpty />
          ) : (
            <GraphCanvas
              directionById={graph.directionById}
              graph={graph.displayGraph}
              hiddenCount={graph.hiddenTotal}
              impactActive={graph.impactActive}
              impactEdgeIds={graph.impactEdgeIds}
              onExplore={exploreById}
              onSelect={selectEntity}
              onToggleGroup={toggleGroup}
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
        projectId={project.id}
      />
    </div>
  )
}

function ReviewEmpty() {
  return (
    <div className="grid h-full place-items-center px-6 text-center">
      <div className="max-w-sm">
        <div className="mx-auto grid size-14 place-items-center rounded-2xl border border-amber-400/30 bg-amber-400/10 text-amber-300">
          <Icon className="size-5" name="diff" />
        </div>
        <h3 className="mt-5 text-sm font-semibold text-foreground">No uncommitted changes</h3>
        <p className="mt-2 text-xs leading-5 text-muted">
          Edit a tracked file in this repository and its blast radius will appear here.
        </p>
      </div>
    </div>
  )
}
