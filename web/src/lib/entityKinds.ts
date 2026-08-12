import type { IconName } from '../components/Icon'

// Single source of truth for how each entity kind is classified and rendered.
// Previously this knowledge was duplicated across three lists (tone classes and
// icon switch in AtlasNode, meaningful/reference sets in graph filtering).
export interface KindMeta {
  // Tailwind border + text classes conveying the kind's tone.
  tone: string
  icon: IconName
  // Architectural kinds shown on the canvas by default.
  meaningful: boolean
  // External/unresolved "reference noise" hidden unless opted in.
  reference: boolean
}

export const KIND_META: Record<string, KindMeta> = {
  package: {
    tone: 'border-primary/45 text-primary',
    icon: 'folder',
    meaningful: true,
    reference: false,
  },
  module: {
    tone: 'border-primary/45 text-primary',
    icon: 'folder',
    meaningful: true,
    reference: false,
  },
  file: {
    tone: 'border-border-strong text-muted',
    icon: 'file',
    meaningful: true,
    reference: false,
  },
  function: {
    tone: 'border-sky-400/40 text-sky-300',
    icon: 'code',
    meaningful: true,
    reference: false,
  },
  method: {
    tone: 'border-sky-400/40 text-sky-300',
    icon: 'code',
    meaningful: true,
    reference: false,
  },
  class: {
    tone: 'border-violet-400/40 text-violet-300',
    icon: 'layers',
    meaningful: true,
    reference: false,
  },
  struct: {
    tone: 'border-violet-400/40 text-violet-300',
    icon: 'layers',
    meaningful: true,
    reference: false,
  },
  interface: {
    tone: 'border-violet-400/40 text-violet-300',
    icon: 'layers',
    meaningful: true,
    reference: false,
  },
  route: {
    tone: 'border-amber-400/45 text-amber-300',
    icon: 'route',
    meaningful: true,
    reference: false,
  },
  external_symbol: {
    tone: 'border-rose-400/35 text-rose-300',
    icon: 'external',
    meaningful: false,
    reference: true,
  },
  unresolved_symbol: {
    tone: 'border-border-strong text-dim',
    icon: 'external',
    meaningful: false,
    reference: true,
  },
}

const FALLBACK: KindMeta = {
  tone: 'border-border-strong text-muted',
  icon: 'code',
  meaningful: false,
  reference: false,
}

export function kindMeta(kind: string): KindMeta {
  return KIND_META[kind] ?? FALLBACK
}

export const MEANINGFUL_KINDS = new Set(
  Object.entries(KIND_META)
    .filter(([, meta]) => meta.meaningful)
    .map(([kind]) => kind),
)

export const REFERENCE_KINDS = new Set(
  Object.entries(KIND_META)
    .filter(([, meta]) => meta.reference)
    .map(([kind]) => kind),
)

// Kinds that own execution edges, so flow/impact lenses can start from them.
export const TRACEABLE_KINDS = new Set(['function', 'method', 'route'])

export function isTraceable(kind?: string): boolean {
  return Boolean(kind) && TRACEABLE_KINDS.has(kind as string)
}
