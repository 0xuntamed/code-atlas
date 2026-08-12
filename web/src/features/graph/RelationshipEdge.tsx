import { memo } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  getSmoothStepPath,
  type Edge,
  type EdgeProps,
} from '@xyflow/react'

export type RelationshipEdgeData = {
  label?: string
}

export type RelationshipFlowEdge = Edge<RelationshipEdgeData, 'relationship'>

export const RelationshipEdge = memo(function RelationshipEdge({
  data,
  markerEnd,
  sourcePosition,
  sourceX,
  sourceY,
  style,
  targetPosition,
  targetX,
  targetY,
}: EdgeProps<RelationshipFlowEdge>) {
  const [path, labelX, labelY] = getSmoothStepPath({
    sourcePosition,
    sourceX,
    sourceY,
    targetPosition,
    targetX,
    targetY,
    borderRadius: 12,
    offset: 28,
  })

  return (
    <>
      <BaseEdge markerEnd={markerEnd} path={path} style={style} />
      {data?.label ? (
        <EdgeLabelRenderer>
          <span
            className="pointer-events-none absolute rounded-full border border-border/80 bg-panel-raised/95 px-2 py-1 font-mono text-[8px] font-semibold uppercase tracking-[0.11em] text-muted shadow-sm"
            style={{
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            }}
          >
            {data.label}
          </span>
        </EdgeLabelRenderer>
      ) : null}
    </>
  )
})
