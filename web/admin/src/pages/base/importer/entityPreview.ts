// 业务实体批量导入:行校验/类型矫正/模板生成(纯函数,UI 与单测共用)。
// 行来源统一为 JSON 数组(或 {rows:[...]} 信封),Excel 通道解析后走同一链路。
import type { EntityDef } from './entities'

export type RowResult =
  | { ok: true; body: Record<string, unknown> }
  | { ok: false; field: string }

export type EntityParseResult =
  | { ok: true; rows: Array<Record<string, unknown>>; rawRows: unknown[] }
  | { ok: false; reason: 'notArray' | 'badRow' | 'tooMany'; row?: number; field?: string; count?: number }

/** 单次导入行数上限:防万行级请求打满创建端点(社区实践:硬上限+分批)。 */
export const MAX_IMPORT_ROWS = 500

/** 单行矫正:trim 字符串、number 转数值、list 按中英逗号/分号拆数组;可选列空值剔除。 */
export function coerceEntityRow(def: EntityDef, row: Record<string, unknown>): RowResult {
  const body: Record<string, unknown> = {}
  for (const col of def.columns) {
    const raw = row[col.key]
    if (col.type === 'list') {
      const arr = toList(raw)
      if (arr) body[col.key] = arr
      continue
    }
    const s = raw === undefined || raw === null ? '' : String(raw).trim()
    if (s === '') {
      if (col.required) return { ok: false, field: col.key }
      continue
    }
    if (col.type === 'number') {
      const n = Number(s)
      if (!Number.isFinite(n)) return { ok: false, field: col.key }
      body[col.key] = n
      continue
    }
    body[col.key] = s
  }
  return { ok: true, body }
}

/** list 列:数组直通;字符串按中英逗号/分号拆分并 trim;空值返回 null(剔除)。 */
function toList(raw: unknown): string[] | null {
  if (Array.isArray(raw)) return raw.map((v) => String(v).trim()).filter(Boolean)
  if (typeof raw === 'string' || typeof raw === 'number') {
    const parts = String(raw).split(/[,;，；]/).map((v) => v.trim()).filter(Boolean)
    return parts.length ? parts : null
  }
  return null
}

/** 顶层形状校验 + 逐行必填检查;行号 1 起算。 */
export function parseEntityRows(def: EntityDef, value: unknown): EntityParseResult {
  const arr = Array.isArray(value) ? value : rowsEnvelope(value)
  if (!arr) return { ok: false, reason: 'notArray' }
  if (arr.length > MAX_IMPORT_ROWS) return { ok: false, reason: 'tooMany', count: arr.length }
  const rows: Array<Record<string, unknown>> = []
  for (let i = 0; i < arr.length; i++) {
    const it = arr[i]
    if (typeof it !== 'object' || it === null || Array.isArray(it)) {
      return { ok: false, reason: 'badRow', row: i + 1 }
    }
    const r = coerceEntityRow(def, it as Record<string, unknown>)
    if (!r.ok) return { ok: false, reason: 'badRow', row: i + 1, field: r.field }
    rows.push(r.body)
  }
  return { ok: true, rows, rawRows: arr }
}

/** 容错:也接受 {"rows":[...]} 信封形态。 */
function rowsEnvelope(v: unknown): unknown[] | null {
  if (typeof v !== 'object' || v === null || Array.isArray(v)) return null
  const rows = (v as Record<string, unknown>).rows
  return Array.isArray(rows) ? rows : null
}

/** JSON 模板:样例行原样序列化(list 列以字符串形态展示,与 Excel 模板一致)。 */
export function entityTemplateJson(def: EntityDef): string {
  return JSON.stringify(def.samples, null, 2)
}
