import { memo } from 'react'
import { Handle, Position, type Node, type NodeProps } from '@xyflow/react'
import { Icon, type IconName } from '../../components/Icon'
import { entityKindLabel } from '../../lib/graph'
import type { Entity } from '../../types'

export type AtlasFlowNode = Node<{ entity: Entity; compact: boolean }, 'atlas'>

const toneClasses: Record<string, string> = {
  route: 'border-amber-400/30 bg-amber-400/[0.07] text-amber-200',
  function: 'border-sky-400/25 bg-sky-400/[0.06] text-sky-200',
  method: 'border-sky-400/25 bg-sky-400/[0.06] text-sky-200',
  class: 'border-violet-400/25 bg-violet-400/[0.06] text-violet-200',
  struct: 'border-violet-400/25 bg-violet-400/[0.06] text-violet-200',
  interface: 'border-violet-400/25 bg-violet-400/[0.06] text-violet-200',
  module: 'border-primary/30 bg-primary/[0.07] text-primary',
  package: 'border-primary/30 bg-primary/[0.07] text-primary',
  file: 'border-border-strong bg-panel-raised text-muted',
  external_symbol: 'border-rose-400/20 bg-rose-400/[0.05] text-rose-200',
  unresolved_symbol: 'border-border bg-canvas-raised text-dim',
}

export const AtlasNode = memo(function AtlasNode({ data, selected }: NodeProps<AtlasFlowNode>) {
  const { entity, compact } = data
  const expandable = entity.kind === 'module' || entity.kind === 'file'
  const stats = moduleStats(entity)
  const icon = iconForKind(entity.kind)

  return (
    <article
      className={`group relative w-[214px] rounded-card border px-3.5 py-3 shadow-[0_12px_35px_oklch(0.03_0.01_244/0.28)] transition-[border-color,box-shadow,opacity,transform] ${toneClasses[entity.kind] ?? toneClasses.file} ${selected ? 'border-primary ring-2 ring-primary/20 shadow-[0_18px_45px_oklch(0.03_0.01_244/0.55)]' : ''} ${entity.distance && entity.distance > 3 ? 'opacity-70' : ''} ${compact ? 'py-2.5' : ''}`}
    >
      <Handle
        className="!size-2.5 !border-2 !border-canvas !bg-border-strong"
        position={Position.Left}
        type="target"
      />
      <div className="flex items-center gap-2">
        <span className="grid size-7 shrink-0 place-items-center rounded-lg border border-current/15 bg-current/8">
          <Icon className="size-3.5" name={icon} />
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
        {expandable ? (
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
        className="!size-2.5 !border-2 !border-canvas !bg-border-strong"
        position={Position.Right}
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

function iconForKind(kind: string): IconName {
  if (kind === 'route') return 'route'
  if (kind === 'module' || kind === 'package') return 'folder'
  if (kind === 'file') return 'file'
  if (kind === 'external_symbol' || kind === 'unresolved_symbol') return 'external'
  if (kind === 'class' || kind === 'struct' || kind === 'interface') return 'layers'
  return 'code'
}
