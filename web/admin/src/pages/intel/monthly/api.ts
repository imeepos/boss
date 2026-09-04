// 月度填报请求层。列表/汇总/区域/单行 upsert 走 apiFetch(envelope 解包);
// 导入需区分 code=0 与 42200(行级失败时 ImportResult 仍随信封带回)、导出是裸 CSV
// (非 envelope)且文件名在 Content-Disposition,两者直用 fetch 携 Bearer。
import { apiFetch, apiBaseUrl, getAuthToken } from '../../../api/client'
import { ApiError, type Envelope } from '../../../api/envelope'
import type { ImportResult, MonthlyRegion, MonthlyRow, MonthlySummary, PageResult, TableKey } from './types'

/** 行级失败业务码(成功计数与错误清单并存,页面不得清空已导入数)。 */
export const CODE_PARTIAL = 42200

export interface MonthlyListQuery {
  month: string
  region: string
  page: number
  pageSize: number
}

function authHeaders(): Record<string, string> {
  const h: Record<string, string> = {}
  const token = getAuthToken()
  if (token) h.Authorization = 'Bearer ' + token
  return h
}

export function fetchRegions(): Promise<MonthlyRegion[]> {
  return apiFetch<{ items: MonthlyRegion[] }>('/monthly/regions').then((d) => d?.items ?? [])
}

export function fetchSummary(month: string): Promise<MonthlySummary | null> {
  return apiFetch<MonthlySummary>('/monthly/summary', { query: { month } })
}

export function fetchRows(table: TableKey, q: MonthlyListQuery): Promise<PageResult<MonthlyRow> | null> {
  return apiFetch<PageResult<MonthlyRow>>('/monthly/' + table, {
    query: { month: q.month, region: q.region, page: q.page, pageSize: q.pageSize },
  })
}

/** 单行 upsert:请求体仅非派生字段,派生列服务端计算,传入无效。 */
export function upsertMonthlyRow(
  table: TableKey,
  body: { month: string; region: string; values: Record<string, number> },
): Promise<unknown> {
  return apiFetch('/monthly/' + table, {
    method: 'PUT',
    body: { month: body.month, region: body.region, ...body.values },
  })
}

export interface ImportOutcome {
  /** 0=全部成功;42200=存在行级失败。 */
  code: number
  result: ImportResult
}

/** CSV 导入(multipart,字段名 file;UTF-8 with BOM)。 */
export async function importMonthlyCsv(table: TableKey, file: File): Promise<ImportOutcome> {
  const fd = new FormData()
  fd.append('file', file)
  const res = await fetch(apiBaseUrl() + '/monthly/' + table + '/import', {
    method: 'POST',
    headers: authHeaders(),
    body: fd,
  })
  if (!res.ok) throw new ApiError(res.status, 'HTTP ' + res.status)
  const env = (await res.json()) as Envelope<ImportResult>
  const partial = env.code === CODE_PARTIAL
  if ((!partial && env.code !== 0) || !env.data) throw new ApiError(env.code, env.msg || 'import failed')
  return { code: env.code, result: env.data }
}

/** CSV 导出:裸 CSV(BOM+CRLF,非 envelope),按 Content-Disposition filename 落盘并返回文件名。 */
export async function exportMonthlyCsv(table: TableKey, month: string): Promise<string> {
  const q = month ? '?month=' + encodeURIComponent(month) : ''
  const res = await fetch(apiBaseUrl() + '/monthly/' + table + '/export' + q, { headers: authHeaders() })
  if (!res.ok) throw new ApiError(res.status, 'HTTP ' + res.status)
  const dispo = res.headers.get('Content-Disposition') ?? ''
  const hit = /filename="([^"]+)"/.exec(dispo)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = hit ? hit[1] : 'monthly-' + table + '.csv'
  a.click()
  URL.revokeObjectURL(url)
  return a.download
}
