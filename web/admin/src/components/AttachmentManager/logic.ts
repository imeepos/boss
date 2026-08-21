// AttachmentManager 纯逻辑:字节格式化 / 查询参数组装 / 选择集操作(可单测)。
import type { AttachmentListParams } from '../../api/attachments'

/** 文件大小展示:B/KB/MB/GB,一位小数(≥1 单位才取整)。 */
export function fmtBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) return '—'
  if (n < 1024) return `${n} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let v = n
  let i = -1
  do {
    v /= 1024
    i++
  } while (v >= 1024 && i < units.length - 1)
  return `${v >= 100 ? Math.round(v) : v.toFixed(1)} ${units[i]}`
}

export interface ListQueryInput {
  uploaderType: string
  uploaderId: number | null
  keyword: string
  page: number
  pageSize: number
}

/** 组装列表查询参数:空筛选剔除、page→offset(类型/ id 成对,见后端契约)。 */
export function toListQuery(q: ListQueryInput): AttachmentListParams {
  const out: AttachmentListParams = {
    limit: q.pageSize,
    offset: (q.page - 1) * q.pageSize,
  }
  const kw = q.keyword.trim()
  if (kw) out.keyword = kw
  // 上传者类型与 id 必须成对出现(后端契约:只传一侧拒绝)。
  if (q.uploaderType && q.uploaderId && q.uploaderId > 0) {
    out.uploaderType = q.uploaderType
    out.uploaderId = q.uploaderId
  }
  return out
}

/** 复选集增删:保持原顺序,不重复。 */
export function toggleSelection(current: number[], id: number, checked: boolean): number[] {
  if (checked) {
    return current.includes(id) ? current : [...current, id]
  }
  return current.filter((it) => it !== id)
}

/** 上传前客户端校验:超限文件名列表(后端上限 32MB,提前拦截省流量)。 */
export const MAX_UPLOAD_BYTES = 32 * 1024 * 1024

export function oversizeFiles(files: File[]): string[] {
  return files.filter((f) => f.size > MAX_UPLOAD_BYTES).map((f) => f.name)
}
