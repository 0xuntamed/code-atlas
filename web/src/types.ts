export type SourceType = 'local' | 'git'
export type ProjectStatus = 'queued' | 'analyzing' | 'ready' | 'failed'
export type GraphView = 'architecture' | 'flow' | 'impact'

export interface AnalysisRun {
  id: string
  projectId: string
  status: string
  stage: string
  completed: number
  total: number
  message?: string
  errorMessage?: string
  startedAt?: string
  completedAt?: string
  createdAt?: string
}

export interface Project {
  id: string
  name: string
  sourceType: SourceType
  rootPath: string
  remoteUrl?: string
  currentCommit?: string
  managedClone: boolean
  status: ProjectStatus
  activeRunId?: string
  latestRun?: AnalysisRun
  createdAt?: string
  updatedAt?: string
}

export interface SourceRange {
  startLine: number
  startColumn: number
  endLine: number
  endColumn: number
}

export interface Entity {
  id: string
  projectId: string
  runId: string
  fileId?: string
  kind: string
  name: string
  qualifiedName: string
  language?: string
  range: SourceRange
  metadata?: Record<string, unknown>
  distance?: number
  isTest?: boolean
}

export interface Relationship {
  id: string
  projectId: string
  runId: string
  source: string
  target: string
  kind: string
  confidence: number
  evidenceFileId?: string
  range: SourceRange
  resolution: string
  metadata?: Record<string, unknown>
}

export interface GraphResponse {
  nodes: Entity[]
  edges: Relationship[]
  rootId?: string
  truncated: boolean
  limit: number
  maxDepth?: number
}

export interface ArchitectureCrumb {
  id: string
  name: string
  kind: string
}

export interface FileRecord {
  id: string
  path: string
  language?: string
  classification: string
  ignoreReason?: string
  isDirectory: boolean
  sizeBytes: number
  isTest: boolean
}

export interface SourceEvidence {
  entityId: string
  filePath: string
  language: string
  startLine: number
  endLine: number
  code: string
}

export interface GraphFilters {
  showTests: boolean
  showReferences: boolean
}
