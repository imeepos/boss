// 客户端版本发布接口:对接 /client-releases*(fields.md 8F;上传为 multipart,后端上限 256MB)。
import { apiFetch } from './client'

export interface ClientReleaseDTO {
  id: number
  app: 'user' | 'worker'
  platform: string
  version: string
  versionCode: number
  minSupportedCode: number
  notes: string
  force: boolean
  status: 'DRAFT' | 'GRAY' | 'PUBLISHED' | 'ROLLED_BACK'
  rolloutPercent: number
  whitelistIds: number[]
  apkObjectKey: string
  apkSize: number
  sha256: string
  createdAt: string
  updatedAt: string
}

export interface ReleaseUploadInput {
  app: 'user' | 'worker'
  version: string
  versionCode: number
  minSupportedCode: number
  notes: string
  file: File
}

/** 上传 APK 并创建发版记录(初始 DRAFT,再经 PATCH 走灰度/全量)。 */
export function uploadRelease(input: ReleaseUploadInput): Promise<ClientReleaseDTO | null> {
  const form = new FormData()
  form.append('file', input.file)
  form.append('app', input.app)
  form.append('version', input.version)
  form.append('versionCode', String(input.versionCode))
  if (input.minSupportedCode > 0) form.append('minSupportedCode', String(input.minSupportedCode))
  if (input.notes) form.append('notes', input.notes)
  return apiFetch<ClientReleaseDTO>('/client-releases', { method: 'POST', body: form })
}

export interface ReleasePatchInput {
  version?: string
  versionCode?: number
  minSupportedCode?: number
  notes?: string
  force?: boolean
  status?: ClientReleaseDTO['status']
  rolloutPercent?: number
  whitelistIds?: number[]
}

export function patchRelease(id: number, patch: ReleasePatchInput): Promise<ClientReleaseDTO | null> {
  return apiFetch<ClientReleaseDTO>(`/client-releases/${id}`, { method: 'PATCH', body: patch })
}

export function listReleases(app?: string): Promise<ClientReleaseDTO[] | null> {
  return apiFetch<{ items: ClientReleaseDTO[] }>('/client-releases', app ? { query: { app } } : {})
    .then((d) => d?.items ?? [])
    .catch(() => null)
}
