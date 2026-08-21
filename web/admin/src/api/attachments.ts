// 附件域接口:对接 /attachments*(信封解包/ token 注入由 apiFetch 统一处理)。
import { apiFetch } from './client'

export type UploaderType = 'account' | 'worker' | 'customer'

export interface AttachmentDTO {
  id: number
  objectKey: string
  fileName: string
  contentType: string
  sizeBytes: number
  uploaderType: UploaderType
  uploaderId: number
  createdAt: string
}

export interface AttachmentListParams {
  /** 指定上传者(类型+id 成对);缺省 = 当前登录身份。 */
  uploaderType?: string
  uploaderId?: number
  /** 文件名关键词(ILIKE)。 */
  keyword?: string
  limit?: number
  offset?: number
}

export interface AttachmentListResult {
  items: AttachmentDTO[] | null
  total: number
}

/** 上传单文件(multipart 字段 file,后端上限 32MB)。 */
export function uploadAttachment(file: File): Promise<AttachmentDTO | null> {
  const form = new FormData()
  form.append('file', file)
  return apiFetch<AttachmentDTO>('/attachments/upload', { method: 'POST', body: form })
}

/** 分页查询附件(新→旧,不含已删);data 为空时返回空集。 */
export async function listAttachments(params: AttachmentListParams): Promise<AttachmentListResult> {
  const data = await apiFetch<AttachmentListResult>('/attachments', {
    query: {
      uploaderType: params.uploaderType,
      uploaderId: params.uploaderId,
      keyword: params.keyword,
      limit: params.limit,
      offset: params.offset,
    },
  })
  return data ?? { items: [], total: 0 }
}

/** 软删除附件(置 deleted_at,MinIO 对象保留)。 */
export function deleteAttachment(id: number): Promise<null> {
  return apiFetch<null>(`/attachments/${id}`, { method: 'DELETE' })
}

/** 按 id 批量查(选择回显场景,后端去重、滤已删)。 */
export async function getAttachmentsByIds(ids: number[]): Promise<{ items: AttachmentDTO[] }> {
  const data = await apiFetch<{ items: AttachmentDTO[] | null }>('/attachments/batch-get', {
    method: 'POST',
    body: { ids },
  })
  return { items: data?.items ?? [] }
}
