// Excel(.xlsx) 导入支持:浏览器端解析为与 JSON 通道一致的载荷对象(后端零改动)。
// Sheet 约定:addr=单 sheet 首行表头 path/name/countryCode/adminCode;
// geo=sheet 名精确对应 countries/countryNames/subdivisions/subdivisionNames(至少一个)。
// 译名列 locale/name/nameType 在 Excel 中平铺,转换时组装回嵌套 name 对象。
import * as XLSX from 'xlsx'
import type { ImportKind } from './preview'
import type { EntityDef } from './entities'

export type ExcelParseResult =
  | { ok: true; value: unknown }
  | { ok: false; reason: 'noSheet' | 'badHeader' | 'badRow'; sheet?: string; row?: number }

/** 文件是否走 Excel 通道(扩展名优先,mime 兜底)。 */
export function isExcelFile(name: string, mime: string): boolean {
  return /\.xlsx$/i.test(name) || mime === 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
}

type Cells = Array<string | number | boolean>

/** 读 workbook 首 sheet 为二维数组(首行表头,空行跳过,空单元格兜底 '')。 */
function sheetToAoA(buf: ArrayBuffer): { aoa: Cells[]; name: string } {
  const wb = XLSX.read(buf, { type: 'array' })
  const name = wb.SheetNames[0]
  const ws = wb.Sheets[name]
  const aoa = XLSX.utils.sheet_to_json<Cells>(ws, { header: 1, raw: true, defval: '' })
  return { aoa, name }
}

/** 表头行 → 列名到下标映射;缺必需列返回 null。 */
function headerIndex(header: Cells, required: string[]): Map<string, number> | null {
  const map = new Map<string, number>()
  header.forEach((c, i) => map.set(String(c).trim(), i))
  return required.every((k) => map.has(k)) ? map : null
}

const str = (v: unknown): string => (v === undefined || v === null ? '' : String(v).trim())
const num = (v: unknown): number | '' => (typeof v === 'number' && Number.isFinite(v) ? v : '')

/** aoa(含表头) → 行对象数组;rowErr 带行列定位。 */
function toObjects(aoa: Cells[], required: string[], optional: string[]): {
  objs: Array<Record<string, string | number>>
  badRow?: number
  badHeader?: boolean
} {
  if (aoa.length === 0) return { objs: [], badHeader: true }
  const idx = headerIndex(aoa[0], required)
  if (!idx) return { objs: [], badHeader: true }
  const keys = [...required, ...optional]
  const objs: Array<Record<string, string | number>> = []
  for (let r = 1; r < aoa.length; r++) {
    const row = aoa[r]
    if (row.every((c) => str(c) === '')) continue
    const o: Record<string, string | number> = {}
    for (const k of keys) {
      const i = idx.get(k)
      if (i === undefined) continue
      o[k] = k === 'level' ? num(row[i]) : str(row[i])
    }
    if (required.some((k) => o[k] === '' || o[k] === undefined)) return { objs, badRow: r + 1 }
    objs.push(o)
  }
  return { objs }
}

/** addr 面板:单 sheet → [{path,name,countryCode,adminCode}]。 */
function parseAddr(buf: ArrayBuffer): ExcelParseResult {
  const { aoa, name } = sheetToAoA(buf)
  const { objs, badRow, badHeader } = toObjects(
    aoa, ['path', 'name'], ['countryCode', 'adminCode'],
  )
  if (badHeader) return { ok: false, reason: 'badHeader', sheet: name }
  if (badRow !== undefined) return { ok: false, reason: 'badRow', sheet: name, row: badRow }
  return { ok: true, value: objs }
}

const GEO_SHEETS = ['countries', 'countryNames', 'subdivisions', 'subdivisionNames'] as const

/** geo 面板:4 个命名 sheet → 与 JSON 载荷同构对象;译名列平铺转嵌套 name。 */
function parseGeo(buf: ArrayBuffer): ExcelParseResult {
  const wb = XLSX.read(buf, { type: 'array' })
  const out: Record<string, unknown> = {}
  for (const s of GEO_SHEETS) {
    if (!wb.SheetNames.includes(s)) continue
    const aoa = XLSX.utils.sheet_to_json<Cells>(wb.Sheets[s], { header: 1, raw: true, defval: '' })
    const isCountry = s === 'countries'
    const isSub = s === 'subdivisions'
    const required = isCountry
      ? ['alpha2', 'alpha3', 'numericCode', 'shortName', 'status', 'continentCode']
      : isSub ? ['code', 'countryCode', 'level', 'category']
        : [s === 'countryNames' ? 'countryCode' : 'subdivisionCode', 'locale', 'name', 'nameType']
    const { objs, badRow, badHeader } = toObjects(aoa, required, [])
    if (badHeader) return { ok: false, reason: 'badHeader', sheet: s }
    if (badRow !== undefined) return { ok: false, reason: 'badRow', sheet: s, row: badRow }
    if (isCountry || isSub) {
      out[s] = objs
    } else {
      // 译名:name.locale/name.name/name.nameType 由平铺列组装(契约 docs/contract/fields.md 1.5.1)。
      const owner = s === 'countryNames' ? 'countryCode' : 'subdivisionCode'
      out[s] = objs.map((o) => ({
        [owner]: o[owner],
        name: { locale: o.locale, name: o.name, nameType: o.nameType },
      }))
    }
  }
  if (Object.keys(out).length === 0) return { ok: false, reason: 'noSheet' }
  return { ok: true, value: out }
}

/** 解析入口:按面板类型转换为 JSON 通道同构载荷。 */
export function parseExcel(kind: ImportKind, buf: ArrayBuffer): ExcelParseResult {
  return kind === 'addr' ? parseAddr(buf) : parseGeo(buf)
}

/** 业务实体面板:单 sheet 首行表头=实体列 key → 行对象数组(类型矫正交给 entityPreview)。 */
export function parseEntityExcel(def: EntityDef, buf: ArrayBuffer): ExcelParseResult {
  const { aoa, name } = sheetToAoA(buf)
  const keys = def.columns.map((c) => c.key)
  const idx = aoa.length > 0 ? headerIndex(aoa[0], keys) : null
  if (!idx) return { ok: false, reason: 'badHeader', sheet: name }
  const objs: Array<Record<string, string | number>> = []
  for (let r = 1; r < aoa.length; r++) {
    const row = aoa[r]
    if (row.every((c) => str(c) === '')) continue
    const o: Record<string, string | number> = {}
    for (const k of keys) o[k] = str(row[idx.get(k) as number])
    objs.push(o)
  }
  if (objs.length === 0) return { ok: false, reason: 'badRow', sheet: name, row: 2 }
  return { ok: true, value: objs }
}

/** 业务实体 xlsx 模板:sheet 名=kind,表头=列 key,样例行与 JSON 模板一致。 */
export function entityExcelTemplate(def: EntityDef): ArrayBuffer {
  const header = def.columns.map((c) => c.key)
  const rows = def.samples.map((s) => header.map((k) => {
    const v = s[k]
    return v === undefined ? '' : Array.isArray(v) ? v.join(',') : v
  }))
  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([header, ...rows]), def.kind)
  return XLSX.write(wb, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer
}

/** 生成 xlsx 模板(表头 + 样例行,与解析约定一致)。 */
export function excelTemplate(kind: ImportKind): ArrayBuffer {
  const wb = XLSX.utils.book_new()
  if (kind === 'addr') {
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([
      ['path', 'name', 'countryCode', 'adminCode'],
      ['cn', 'China', 'CN', ''],
      ['cn.gd', 'Guangdong', 'CN', ''],
    ]), 'addresses')
  } else {
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([
      ['alpha2', 'alpha3', 'numericCode', 'shortName', 'status', 'continentCode'],
      ['CN', 'CHN', '156', 'China', 'INDEPENDENT', 'AS'],
    ]), 'countries')
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([
      ['countryCode', 'locale', 'name', 'nameType'],
      ['CN', 'zh-Hans', '中国', 'STANDARD'],
    ]), 'countryNames')
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([
      ['code', 'countryCode', 'level', 'category'],
      ['CN-GD', 'CN', 1, 'province'],
    ]), 'subdivisions')
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([
      ['subdivisionCode', 'locale', 'name', 'nameType'],
      ['CN-GD', 'zh-Hans', '广东', 'STANDARD'],
    ]), 'subdivisionNames')
  }
  return XLSX.write(wb, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer
}
