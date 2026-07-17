import { memo } from 'react'
import { Handle, Position, type Node, type NodeProps } from '@xyflow/react'
import type { Entity } from '../../types'

export type AtlasFlowNode = Node<
  {
    entity: Entity
    compact: boolean
  },
  'atlas'
>

const kindTone: Record<string, string> = {
  route: 'amber',
  function: 'blue',
  method: 'blue',
  class: 'violet',
  struct: 'violet',
  interface: 'violet',
  module: 'teal',
  file: 'slate',
  external_symbol: 'rose',
  unresolved_symbol: 'muted',
}

export const AtlasNode = memo(function AtlasNode({ data, selected }: NodeProps<AtlasFlowNode>) {
  const { entity, compact } = data
  const expandable = entity.kind === 'module' || entity.kind === 'file'
  const stats = moduleStats(entity)

  return (
    <article
      className={[
        'atlas-node',
        `tone-${kindTone[entity.kind] ?? 'slate'}`,
        selected ? 'is-selected' : '',
        entity.distance ? `distance-${Math.min(entity.distance, 4)}` : '',
        compact ? 'compact-node' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <Handle type="target" position={Position.Left} />
      <div className="node-heading">
        <span>{entity.kind.replace('_', ' ')}</span>
        {expandable && <i aria-label="Expandable">open</i>}
      </div>
      <strong title={entity.qualifiedName}>{entity.name}</strong>
      {stats ? (
        <small>{stats}</small>
      ) : (
        <small>
          {entity.language || (entity.kind === 'external_symbol' ? 'external' : 'derived')}
        </small>
      )}
      <Handle type="source" position={Position.Right} />
    </article>
  )
})

function moduleStats(entity: Entity): string | undefined {
  if (entity.kind !== 'module' || typeof entity.metadata?.fileCount !== 'number') {
    return undefined
  }
  const files = metadataNumber(entity, 'fileCount')
  const symbols = metadataNumber(entity, 'symbolCount')
  const routes = metadataNumber(entity, 'routeCount')
  const parts = [entity.qualifiedName, `${files} files`, `${symbols} symbols`]
  if (routes > 0) {
    parts.push(`${routes} routes`)
  }
  return parts.join(' · ')
}

function metadataNumber(entity: Entity, key: string): number {
  const value = entity.metadata?.[key]
  return typeof value === 'number' ? value : 0
}
