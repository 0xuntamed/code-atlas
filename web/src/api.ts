import type {
  ChangesResponse,
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

async function request<T>(path: string, init?: RequestInit, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    signal,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
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
  projects: (signal?: AbortSignal) =>
    request<{ projects: Project[] }>('/projects', undefined, signal),
  project: (projectId: string, signal?: AbortSignal) =>
    request<Project>(`/projects/${encodeURIComponent(projectId)}`, undefined, signal),
  createProject: (source: { type: SourceType; path?: string; url?: string; ref?: string }) =>
    request<{ projectId: string; analysisRunId: string; status: string }>('/projects', {
      method: 'POST',
      body: JSON.stringify({ source }),
    }),
  deleteProject: (projectId: string) =>
    request<void>(`/projects/${encodeURIComponent(projectId)}`, { method: 'DELETE' }),
  shutdown: () =>
    request<{ status: string }>('/system/shutdown', {
      method: 'POST',
      headers: { 'X-CodeAtlas-Intent': 'shutdown' },
    }),
  analyze: (projectId: string) =>
    request<{ analysisRunId: string }>(`/projects/${encodeURIComponent(projectId)}/analyses`, {
      method: 'POST',
    }),
  architecture: (projectId: string, scopeId = '', signal?: AbortSignal) => {
    const scope = scopeId ? `&scope=${encodeURIComponent(scopeId)}` : ''
    return request<GraphResponse>(
      `/projects/${encodeURIComponent(projectId)}/graph/architecture?limit=80${scope}`,
      undefined,
      signal,
    )
  },
  flow: (projectId: string, entityId: string, signal?: AbortSignal) =>
    request<GraphResponse>(
      `/projects/${encodeURIComponent(projectId)}/flow/${encodeURIComponent(entityId)}?depth=6&limit=120`,
      undefined,
      signal,
    ),
  impact: (projectId: string, entityId: string, signal?: AbortSignal) =>
    request<GraphResponse>(
      `/projects/${encodeURIComponent(projectId)}/impact/${encodeURIComponent(entityId)}?direction=both&depth=4&limit=120`,
      undefined,
      signal,
    ),
  impactMap: (projectId: string, entityId: string, signal?: AbortSignal) =>
    request<GraphResponse>(
      `/projects/${encodeURIComponent(projectId)}/impact-map/${encodeURIComponent(entityId)}?depth=4&limit=160`,
      undefined,
      signal,
    ),
  changes: (projectId: string, signal?: AbortSignal) =>
    request<ChangesResponse>(
      `/projects/${encodeURIComponent(projectId)}/changes`,
      undefined,
      signal,
    ),
  search: (projectId: string, query: string, signal?: AbortSignal) =>
    request<{ entities: Entity[] }>(
      `/projects/${encodeURIComponent(projectId)}/search?q=${encodeURIComponent(query)}`,
      undefined,
      signal,
    ),
  files: (projectId: string, classification = '', limit = 200, signal?: AbortSignal) =>
    request<{ files: FileRecord[] }>(
      `/projects/${encodeURIComponent(projectId)}/files?classification=${encodeURIComponent(classification)}&limit=${limit}`,
      undefined,
      signal,
    ),
  entity: (projectId: string, entityId: string, signal?: AbortSignal) =>
    request<Entity>(
      `/projects/${encodeURIComponent(projectId)}/entities/${encodeURIComponent(entityId)}`,
      undefined,
      signal,
    ),
  source: (projectId: string, entityId: string, signal?: AbortSignal) =>
    request<SourceEvidence>(
      `/projects/${encodeURIComponent(projectId)}/entities/${encodeURIComponent(entityId)}/source`,
      undefined,
      signal,
    ),
}
