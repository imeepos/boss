// AttachmentManager 纯逻辑:字节格式化 / 查询参数组装 / 选择集操作 / 文件分类(可单测)。
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

/** 9 类文件分类 key(顺序与侧栏一致)。 */
export const FILE_CATEGORY_KEYS = [
  'image', 'audio', 'video', 'pdf', 'document',
  'spreadsheet', 'archive', 'code', 'other',
] as const

export type FileCategoryKey = typeof FILE_CATEGORY_KEYS[number]

const EXT_MAP: Record<string, FileCategoryKey> = {
  // 图片
  jpg: 'image', jpeg: 'image', png: 'image', gif: 'image', webp: 'image',
  svg: 'image', bmp: 'image', heic: 'image', ico: 'image',
  // 音频
  mp3: 'audio', wav: 'audio', aac: 'audio', flac: 'audio', ogg: 'audio', m4a: 'audio',
  // 视频
  mp4: 'video', avi: 'video', mov: 'video', mkv: 'video', webm: 'video', flv: 'video', m4v: 'video',
  // PDF
  pdf: 'pdf',
  // 文档
  doc: 'document', docx: 'document', rtf: 'document', odt: 'document', pages: 'document',
  // 表格
  xls: 'spreadsheet', xlsx: 'spreadsheet', csv: 'spreadsheet', ods: 'spreadsheet', numbers: 'spreadsheet',
  // 压缩包
  zip: 'archive', rar: 'archive', '7z': 'archive', tar: 'archive', gz: 'archive', bz2: 'archive',
  'tar.gz': 'archive', 'tar.bz2': 'archive', tgz: 'archive', tbz2: 'archive', xz: 'archive',
  // 代码/数据
  json: 'code', xml: 'code', yaml: 'code', yml: 'code', sql: 'code', sh: 'code',
  js: 'code', ts: 'code', tsx: 'code', jsx: 'code', html: 'code', htm: 'code',
  css: 'code', scss: 'code', less: 'code', md: 'code', py: 'code', go: 'code',
  java: 'code', kt: 'code', c: 'code', cpp: 'code', h: 'code', hpp: 'code', rs: 'code',
  toml: 'code', ini: 'code', env: 'code', properties: 'code', log: 'code',
}

const MIME_MAP: Record<string, FileCategoryKey> = {
  'application/pdf': 'pdf',
  'application/zip': 'archive',
  'application/x-rar-compressed': 'archive',
  'application/x-7z-compressed': 'archive',
  'application/x-tar': 'archive',
  'application/gzip': 'archive',
  'application/x-bzip2': 'archive',
  'application/msword': 'document',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'document',
  'application/rtf': 'document',
  'application/vnd.oasis.opendocument.text': 'document',
  'application/vnd.ms-excel': 'spreadsheet',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'spreadsheet',
  'application/vnd.oasis.opendocument.spreadsheet': 'spreadsheet',
  'text/csv': 'spreadsheet',
}

/** MIME 主类型前缀分类(覆盖 audio/* / video/* / image/* / text/* 中的代码类)。 */
function classifyByMime(mime: string): FileCategoryKey | null {
  const lower = mime.toLowerCase()
  if (lower.startsWith('image/')) return 'image'
  if (lower.startsWith('audio/')) return 'audio'
  if (lower.startsWith('video/')) return 'video'
  if (MIME_MAP[lower]) return MIME_MAP[lower]
  // text/* 里大量代码/数据归 code,csv 已经在 MIME_MAP
  if (lower.startsWith('text/')) return 'code'
  return null
}

/** 从文件名取扩展名(去除 query/fragment,处理双扩展如 file.tar.gz → tar.gz)。 */
function getExt(fileName: string): string {
  const name = fileName.split('?')[0].split('#')[0].split('/').pop() ?? fileName
  const lower = name.toLowerCase()
  // 双扩展兜底(优先取次级)
  const compound = lower.match(/\.([a-z0-9]{1,5})\.([a-z0-9]{1,5})$/)
  if (compound && (compound[2] === 'gz' || compound[2] === 'bz2')) return `${compound[1]}.${compound[2]}`
  const dot = lower.lastIndexOf('.')
  if (dot < 0 || dot === lower.length - 1) return ''
  return lower.slice(dot + 1)
}

/** 把附件归入 9 类之一:优先 MIME,其次扩展名,兜底 'other'。 */
export function classifyAttachment(contentType: string, fileName: string): FileCategoryKey {
  const fromMime = classifyByMime(contentType || '')
  if (fromMime) return fromMime
  const ext = getExt(fileName)
  if (!ext) return 'other'
  return EXT_MAP[ext] ?? 'other'
}

/** 对 items 按当前 category 过滤(客户端过滤,见 spec §"后端契约")。 */
export function filterByCategory<T extends { contentType: string; fileName: string }>(
  items: T[], category: FileCategoryKey | '',
): T[] {
  if (!category) return items
  return items.filter((it) => classifyAttachment(it.contentType, it.fileName) === category)
}

/** 统计 items 中各类数量(用于侧栏 badge 显示当前页分布)。 */
export function countByCategory<T extends { contentType: string; fileName: string }>(
  items: T[],
): Record<FileCategoryKey, number> {
  const counts: Record<FileCategoryKey, number> = {
    image: 0, audio: 0, video: 0, pdf: 0, document: 0,
    spreadsheet: 0, archive: 0, code: 0, other: 0,
  }
  for (const it of items) counts[classifyAttachment(it.contentType, it.fileName)]++
  return counts
}