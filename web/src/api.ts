import type {
  Entity,
  FileRecord,
  GraphResponse,
  Project,
  SourceEvidence,
  SourceType,
} from './types'

export class APIError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message)
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as {
      error?: { code?: string; message?: string }
    } | null
    throw new APIError(
      response.status,
      payload?.error?.code ?? 'request_failed',
      payload?.error?.message ?? response.statusText,
    )
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  projects: () => request<{ projects: Project[] }>('/projects'),
  project: (projectId: string) => request<Project>(`/projects/${projectId}`),
  createProject: (source: { type: SourceType; path?: string; url?: string; ref?: string }) =>
    request<{ projectId: string; analysisRunId: string; status: string }>('/projects', {
      method: 'POST',
      body: JSON.stringify({ source }),
    }),
  deleteProject: (projectId: string) =>
    request<void>(`/projects/${projectId}`, { method: 'DELETE' }),
  shutdown: () =>
    request<{ status: string }>('/system/shutdown', {
      method: 'POST',
      headers: { 'X-CodeAtlas-Intent': 'shutdown' },
    }),
  analyze: (projectId: string) =>
    request<{ analysisRunId: string }>(`/projects/${projectId}/analyses`, { method: 'POST' }),
  architecture: (projectId: string, scopeId = '') => {
    const scope = scopeId ? `&scope=${encodeURIComponent(scopeId)}` : ''
    return request<GraphResponse>(`/projects/${projectId}/graph/architecture?limit=80${scope}`)
  },
  flow: (projectId: string, entityId: string) =>
    request<GraphResponse>(`/projects/${projectId}/flow/${entityId}?depth=6&limit=120`),
  impact: (projectId: string, entityId: string) =>
    request<GraphResponse>(
      `/projects/${projectId}/impact/${entityId}?direction=both&depth=4&limit=120`,
    ),
  search: (projectId: string, query: string) =>
    request<{ entities: Entity[] }>(`/projects/${projectId}/search?q=${encodeURIComponent(query)}`),
  files: (projectId: string, classification = '', limit = 200) =>
    request<{ files: FileRecord[] }>(
      `/projects/${projectId}/files?classification=${encodeURIComponent(classification)}&limit=${limit}`,
    ),
  entity: (projectId: string, entityId: string) =>
    request<Entity>(`/projects/${projectId}/entities/${entityId}`),
  source: (projectId: string, entityId: string) =>
    request<SourceEvidence>(`/projects/${projectId}/entities/${entityId}/source`),
}
