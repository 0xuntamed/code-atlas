import { memo } from 'react'
import { Handle, Position, type Node, type NodeProps } from '@xyflow/react'
import { Icon } from '../../components/Icon'
import { kindMeta } from '../../lib/entityKinds'
import { entityKindLabel } from '../../lib/graph'
import type { Entity, ImpactDirection } from '../../types'

export type AtlasFlowNode = Node<
  {
    entity: Entity
    compact: boolean
    direction: 'horizontal' | 'vertical'
    // Blast-radius role when a node is selected; dimmed nodes are outside it.
    impactDirection?: ImpactDirection
    dimmed?: boolean
  },
  'atlas'
>

// Border + ring per blast-radius role: the changed node, what breaks (dependents),
// and what it relies on (dependencies).
const directionClasses: Record<ImpactDirection, string> = {
  root: '!border-primary ring-2 ring-primary/45',
  dependent: '!border-rose-400 ring-2 ring-rose-400/40',
  dependency: '!border-sky-400 ring-2 ring-sky-400/40',
  both: '!border-violet-400 ring-2 ring-violet-400/40',
}

export const AtlasNode = memo(function AtlasNode({ data, selected }: NodeProps<AtlasFlowNode>) {
  const { entity, compact, direction, impactDirection, dimmed } = data
  const meta = kindMeta(entity.kind)
  const isGroup = entity.metadata?.group === true
  const affectedCount =
    typeof entity.metadata?.affectedCount === 'number' ? entity.metadata.affectedCount : 0
  const expandable = entity.kind === 'module' || entity.kind === 'file'
  const stats = isGroup ? `${affectedCount} affected · expand` : moduleStats(entity)
  const targetPosition = direction === 'vertical' ? Position.Top : Position.Left
  const sourcePosition = direction === 'vertical' ? Position.Bottom : Position.Right
  const emphasis = impactDirection
    ? directionClasses[impactDirection]
    : selected
      ? '!border-primary ring-2 ring-primary/30 shadow-[0_18px_45px_oklch(0.03_0.01_244/0.6)]'
      : ''

  return (
    <article
      className={`group relative w-[214px] rounded-card border bg-panel-raised px-3.5 py-3 shadow-[0_10px_30px_oklch(0.03_0.01_244/0.45)] transition-[border-color,box-shadow,opacity,transform] ${meta.tone} ${emphasis} ${dimmed ? 'opacity-25' : entity.distance && entity.distance > 3 ? 'opacity-75' : ''} ${compact ? 'py-2.5' : ''}`}
    >
      <Handle
        className="!size-2.5 !border-2 !border-canvas !bg-panel-raised transition-colors group-hover:!bg-primary"
        position={targetPosition}
        type="target"
      />
      <div className="flex items-center gap-2">
        <span className="grid size-7 shrink-0 place-items-center rounded-lg border border-current/15 bg-current/8">
          <Icon className="size-3.5" name={meta.icon} />
        </span>
        <div className="min-w-0 flex-1">
          <span className="block text-[8px] font-bold uppercase tracking-[0.16em] opacity-65">
            {entityKindLabel(entity.kind)}
          </span>
          <strong
            className="mt-0.5 block truncate text-xs font-semibold text-foreground"
            title={entity.name}
          >
            {entity.name}
          </strong>
        </div>
        {isGroup ? (
          <Icon
            className="size-3.5 shrink-0 rotate-90 opacity-60 transition-transform group-hover:translate-y-0.5"
            name="chevron-right"
          />
        ) : expandable ? (
          <Icon
            className="size-3.5 shrink-0 opacity-45 transition-transform group-hover:translate-x-0.5"
            name="chevron-right"
          />
        ) : null}
      </div>
      {!compact ? (
        <div className="mt-2 flex items-center gap-2 border-t border-current/10 pt-2 text-[9px] opacity-60">
          <span className="min-w-0 flex-1 truncate" title={entity.qualifiedName}>
            {stats ?? entity.language ?? 'derived metadata'}
          </span>
          {entity.isTest ? (
            <span className="rounded border border-current/20 px-1 py-0.5 font-bold uppercase">
              test
            </span>
          ) : null}
        </div>
      ) : null}
      <Handle
        className="!size-2.5 !border-2 !border-canvas !bg-panel-raised transition-colors group-hover:!bg-primary"
        position={sourcePosition}
        type="source"
      />
    </article>
  )
})

function moduleStats(entity: Entity): string | undefined {
  if (entity.kind !== 'module' || typeof entity.metadata?.fileCount !== 'number') return undefined
  const files = metadataNumber(entity, 'fileCount')
  const symbols = metadataNumber(entity, 'symbolCount')
  const routes = metadataNumber(entity, 'routeCount')
  return `${files} files · ${symbols} symbols${routes > 0 ? ` · ${routes} routes` : ''}`
}

function metadataNumber(entity: Entity, key: string): number {
  const value = entity.metadata?.[key]
  return typeof value === 'number' ? value : 0
}
