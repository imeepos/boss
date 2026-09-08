// 导入载荷解析与预览模型(纯函数,UI 与单测共用;字段口径 docs/contract/fields.md 1.5.1)。
export type ImportKind = 'addr' | 'geo'

export const MAX_BYTES = 10 * 1024 * 1024
export const PREVIEW_ROWS = 5

export type ParseResult = { ok: true; value: unknown } | { ok: false; line?: number; col?: number }

/** JSON.parse;失败时提取失败位置(行:列,供 ErrorBanner 定位)。 */
export function parseJson(text: string): ParseResult {
  try {
    return { ok: true, value: JSON.parse(text) as unknown }
  } catch (e) {
    return { ok: false, ...jsonErrorPosition(e, text) }
  }
}

/** 失败位置提取:Firefox 报 "line N column M";V8 报 "position N"(需按文本换算行:列);
 * 两者皆无(如旧 Safari/源码摘录型报文)返回空对象,由调用方降级为无定位提示。 */
export function jsonErrorPosition(e: unknown, text: string): { line?: number; col?: number } {
  const msg = e instanceof Error ? e.message : ''
  const lineCol = /line (\d+) column (\d+)/.exec(msg)
  if (lineCol) return { line: Number(lineCol[1]), col: Number(lineCol[2]) }
  const posMatch = /position (\d+)/.exec(msg)
  if (!posMatch) {
    const lineOnly = /line (\d+)/.exec(msg)
    return lineOnly ? { line: Number(lineOnly[1]) } : {}
  }
  const pos = Number(posMatch[1])
  if (pos > text.length) return {}
  const lines = text.slice(0, pos).split('\n')
  return { line: lines.length, col: lines[lines.length - 1].length + 1 }
}

export type PreviewReason = 'notArray' | 'notObject' | 'badRow'

export interface PreviewModel {
  /** 实际 POST 体(地址导入包 {rows:[...]},geo 原样)。 */
  body: unknown
  rowCount: number
  columns: string[]
  rows: string[][]
  truncated: boolean
  /** geo 分段计数(labelIndex 对应 i18n geoSections 下标)。 */
  sections: Array<{ labelIndex: number; count: number }>
}

export type PreviewResult =
  | { ok: true; model: PreviewModel }
  | { ok: false; reason: PreviewReason; row?: number }

/** 按面板类型校验形状并生成预览模型。 */
export function buildPreview(kind: ImportKind, value: unknown): PreviewResult {
  return kind === 'addr' ? buildAddrPreview(value) : buildGeoPreview(value)
}

const ADDR_FIELDS = ['path', 'name', 'countryCode', 'adminCode']

function buildAddrPreview(value: unknown): PreviewResult {
  const arr = Array.isArray(value) ? value : rowsEnvelope(value)
  if (!arr) return { ok: false, reason: 'notArray' }
  for (let i = 0; i < arr.length; i++) {
    const it = arr[i] as Record<string, unknown> | null
    if (typeof it?.path !== 'string' || typeof it?.name !== 'string') {
      return { ok: false, reason: 'badRow', row: i + 1 }
    }
  }
  return {
    ok: true,
    model: {
      body: { rows: arr },
      rowCount: arr.length,
      columns: ADDR_FIELDS,
      rows: arr.slice(0, PREVIEW_ROWS).map(rowCells),
      truncated: arr.length > PREVIEW_ROWS,
      sections: [],
    },
  }
}

/** 容错:也接受 {"rows":[...]} 信封形态。 */
function rowsEnvelope(v: unknown): unknown[] | null {
  if (typeof v !== 'object' || v === null || Array.isArray(v)) return null
  const rows = (v as Record<string, unknown>).rows
  return Array.isArray(rows) ? rows : null
}

function rowCells(it: unknown): string[] {
  const o = it as Record<string, unknown>
  return ADDR_FIELDS.map((f) => (o[f] === undefined || o[f] === null ? '' : String(o[f])))
}

const GEO_SECTIONS = ['countries', 'countryNames', 'subdivisions', 'subdivisionNames'] as const

function buildGeoPreview(value: unknown): PreviewResult {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return { ok: false, reason: 'notObject' }
  }
  const o = value as Record<string, unknown>
  if (!GEO_SECTIONS.some((k) => Array.isArray(o[k]))) return { ok: false, reason: 'notObject' }
  const sections = GEO_SECTIONS.map((key, labelIndex) => ({
    labelIndex,
    count: Array.isArray(o[key]) ? (o[key] as unknown[]).length : 0,
  }))
  return {
    ok: true,
    model: {
      body: value,
      rowCount: sections.reduce((s, it) => s + it.count, 0),
      columns: [],
      rows: [],
      truncated: false,
      sections,
    },
  }
}

/** 导入成功结果计数(addr 取 imported;geo 取 countries/subdivisions,与任务记录口径一致)。 */
export function resultCount(kind: ImportKind, res: unknown): {
  total?: number
  countries?: number
  subdivisions?: number
} {
  const o = (res ?? {}) as Record<string, unknown>
  if (kind === 'addr') return { total: num(o.imported) }
  return { countries: num(o.countries), subdivisions: num(o.subdivisions) }
}

function num(v: unknown): number | undefined {
  return typeof v === 'number' ? v : undefined
}

/** 模板下载内容(与各面板输入格式一致的最小合法样例)。 */
export function templateJson(kind: ImportKind): string {
  if (kind === 'addr') {
    return JSON.stringify(
      [
        { path: 'cn', name: 'China', countryCode: 'CN' },
        { path: 'cn.gd', name: 'Guangdong', countryCode: 'CN' },
      ],
      null,
      2,
    )
  }
  return JSON.stringify(
    {
      countries: [{ alpha2: 'CN', alpha3: 'CHN', numericCode: '156', shortName: 'China', status: 'INDEPENDENT', continentCode: 'AS' }],
      countryNames: [{ countryCode: 'CN', name: { locale: 'zh-Hans', name: '中国', nameType: 'STANDARD' } }],
      subdivisions: [{ code: 'CN-GD', countryCode: 'CN', level: 1, category: 'province' }],
      subdivisionNames: [{ subdivisionCode: 'CN-GD', name: { locale: 'zh-Hans', name: '广东', nameType: 'STANDARD' } }],
    },
    null,
    2,
  )
}
