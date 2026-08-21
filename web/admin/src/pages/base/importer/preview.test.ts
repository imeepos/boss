// 导入预览纯函数单测:解析行号定位/rows 包裹/geo 分段计数/结果计数/模板可解析。
import { describe, expect, it } from 'vitest'
import {
  buildPreview, parseJson, resultCount, templateJson,
} from './preview'

describe('parseJson', () => {
  it('合法 JSON 返回值', () => {
    expect(parseJson('[1,2]')).toEqual({ ok: true, value: [1, 2] })
  })
  it('解析失败 ok=false(行号仅 Firefox/旧 V8 报文可提取,新 V8 回落无行号)', () => {
    const r = parseJson('[\n1,\nbad\n]')
    expect(r.ok).toBe(false)
  })
})

describe('buildPreview addr', () => {
  it('数组包裹为 rows 信封并预览前 5 行', () => {
    const rows = Array.from({ length: 7 }, (_, i) => ({ path: `p${i}`, name: `n${i}` }))
    const r = buildPreview('addr', rows)
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.model.body).toEqual({ rows })
    expect(r.model.rowCount).toBe(7)
    expect(r.model.rows).toHaveLength(5)
    expect(r.model.truncated).toBe(true)
  })
  it('接受 {rows:[...]} 信封形态', () => {
    const r = buildPreview('addr', { rows: [{ path: 'gz', name: 'Guangzhou' }] })
    expect(r.ok).toBe(true)
  })
  it('非数组报 notArray', () => {
    expect(buildPreview('addr', { foo: 1 })).toEqual({ ok: false, reason: 'notArray' })
  })
  it('缺 name 的行报 badRow 含行号', () => {
    const r = buildPreview('addr', [{ path: 'a', name: 'x' }, { path: 'b' }])
    expect(r).toEqual({ ok: false, reason: 'badRow', row: 2 })
  })
})

describe('buildPreview geo', () => {
  it('四段计数合计', () => {
    const r = buildPreview('geo', {
      countries: [{}, {}],
      countryNames: [{}],
      subdivisions: [{}, {}, {}],
      subdivisionNames: [],
    })
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.model.rowCount).toBe(6)
    expect(r.model.body).toEqual({
      countries: [{}, {}], countryNames: [{}], subdivisions: [{}, {}, {}], subdivisionNames: [],
    })
  })
  it('无任一合法键报 notObject', () => {
    expect(buildPreview('geo', { other: [] })).toEqual({ ok: false, reason: 'notObject' })
    expect(buildPreview('geo', [1])).toEqual({ ok: false, reason: 'notObject' })
  })
})

describe('resultCount', () => {
  it('addr 取 imported', () => {
    expect(resultCount('addr', { imported: 3 })).toEqual({ total: 3 })
  })
  it('geo 取 countries/subdivisions(与任务记录口径一致)', () => {
    expect(resultCount('geo', { countries: 2, countryNames: 5, subdivisions: 4, subdivisionNames: 9 }))
      .toEqual({ countries: 2, subdivisions: 4 })
  })
})

describe('templateJson', () => {
  it('两份模板均为合法 JSON 且形状正确', () => {
    const addr = parseJson(templateJson('addr'))
    expect(addr.ok).toBe(true)
    expect(buildPreview('addr', addr.ok ? addr.value : null).ok).toBe(true)
    const geo = parseJson(templateJson('geo'))
    expect(geo.ok).toBe(true)
    expect(buildPreview('geo', geo.ok ? geo.value : null).ok).toBe(true)
  })
})
