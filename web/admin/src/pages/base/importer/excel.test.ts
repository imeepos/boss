// Excel 导入解析单测:XLSX 构造 workbook → excelTemplate/parseExcel 回环,覆盖错误分支与预览集成。
import { describe, expect, it } from 'vitest'
import * as XLSX from 'xlsx'
import { excelTemplate, isExcelFile, parseExcel } from './excel'
import { buildPreview } from './preview'

/** aoa → xlsx 二进制(单 sheet)。 */
function xlsxOf(sheetName: string, aoa: unknown[][]): ArrayBuffer {
  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet(aoa), sheetName)
  return XLSX.write(wb, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer
}

/** 多 sheet → xlsx 二进制。 */
function xlsxMulti(sheets: Record<string, unknown[][]>): ArrayBuffer {
  const wb = XLSX.utils.book_new()
  for (const [name, aoa] of Object.entries(sheets)) {
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet(aoa), name)
  }
  return XLSX.write(wb, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer
}

describe('isExcelFile', () => {
  it('按扩展名与 mime 识别', () => {
    expect(isExcelFile('a.xlsx', '')).toBe(true)
    expect(isExcelFile('a.XLSX', '')).toBe(true)
    expect(isExcelFile('a.json', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')).toBe(true)
    expect(isExcelFile('a.json', 'application/json')).toBe(false)
  })
})

describe('parseExcel addr', () => {
  it('表头映射 + 可选列缺省 + 空行跳过', () => {
    const buf = xlsxOf('addresses', [
      ['path', 'name', 'countryCode', 'adminCode'],
      ['cn', 'China', 'CN', ''],
      ['', '', '', ''],
      ['cn.gd', 'Guangdong', 'CN', ''],
    ])
    const r = parseExcel('addr', buf)
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.value).toEqual([
      { path: 'cn', name: 'China', countryCode: 'CN', adminCode: '' },
      { path: 'cn.gd', name: 'Guangdong', countryCode: 'CN', adminCode: '' },
    ])
    const p = buildPreview('addr', r.value)
    expect(p.ok && p.model.rowCount).toBe(2)
  })

  it('表头缺必需列 → badHeader;数据行缺必填 → badRow', () => {
    expect(parseExcel('addr', xlsxOf('s', [['name'], ['x']]))).toMatchObject({ ok: false, reason: 'badHeader' })
    const buf = xlsxOf('s', [
      ['path', 'name'],
      ['cn', 'China'],
      ['', 'Gap'],
    ])
    expect(parseExcel('addr', buf)).toMatchObject({ ok: false, reason: 'badRow', row: 3 })
  })
})

describe('parseExcel geo', () => {
  it('四 sheet 命名匹配,译名列组装回嵌套 name,level 保数值', () => {
    const buf = xlsxMulti({
      countries: [['alpha2', 'alpha3', 'numericCode', 'shortName', 'status', 'continentCode'], ['CN', 'CHN', '156', 'China', 'INDEPENDENT', 'AS']],
      countryNames: [['countryCode', 'locale', 'name', 'nameType'], ['CN', 'zh-Hans', '中国', 'STANDARD']],
      subdivisions: [['code', 'countryCode', 'level', 'category'], ['CN-GD', 'CN', 1, 'province']],
      subdivisionNames: [['subdivisionCode', 'locale', 'name', 'nameType'], ['CN-GD', 'zh-Hans', '广东', 'STANDARD']],
    })
    const r = parseExcel('geo', buf)
    expect(r.ok).toBe(true)
    if (!r.ok) return
    const p = buildPreview('geo', r.value)
    expect(p.ok && p.model.rowCount).toBe(4)
    const v = r.value as Record<string, unknown>
    expect(v.countryNames).toEqual([
      { countryCode: 'CN', name: { locale: 'zh-Hans', name: '中国', nameType: 'STANDARD' } },
    ])
    expect((v.subdivisions as Array<Record<string, unknown>>)[0].level).toBe(1)
  })

  it('无命名 sheet → noSheet;部分 sheet 允许', () => {
    expect(parseExcel('geo', xlsxOf('sheet1', [['a'], ['b']]))).toMatchObject({ ok: false, reason: 'noSheet' })
    const buf = xlsxOf('countries', [
      ['alpha2', 'alpha3', 'numericCode', 'shortName', 'status', 'continentCode'],
      ['CN', 'CHN', '156', 'China', 'INDEPENDENT', 'AS'],
    ])
    const r = parseExcel('geo', buf)
    expect(r.ok).toBe(true)
  })
})

describe('excelTemplate', () => {
  it('模板可被 parseExcel 自身回环解析', () => {
    const addr = parseExcel('addr', excelTemplate('addr'))
    expect(addr.ok).toBe(true)
    expect(addr.ok && buildPreview('addr', addr.value).ok).toBe(true)
    const geo = parseExcel('geo', excelTemplate('geo'))
    expect(geo.ok).toBe(true)
    expect(geo.ok && buildPreview('geo', geo.value).ok).toBe(true)
  })
})
