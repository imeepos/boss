// 导入预览纯函数单测:解析行:列定位/rows 包裹/geo 分段计数/结果计数/模板可解析。
import { describe, expect, it } from 'vitest'
import {
  buildPreview, jsonErrorPosition, parseJson, resultCount, templateJson,
} from './preview'

describe('parseJson', () => {
  it('合法 JSON 返回值', () => {
    expect(parseJson('[1,2]')).toEqual({ ok: true, value: [1, 2] })
  })
  it('解析失败 ok=false(行号仅 Firefox/旧 V8 报文可提取,新 V8 回落无行号)', () => {
    const r = parseJson('[\n1,\nbad\n]')
    expect(r.ok).toBe(false)
  })
  it('解析失败给行:列定位(V8 position 换算)', () => {
    const r = parseJson('{foo}')
    expect(r.ok).toBe(false)
    if (r.ok) return
    // Node V8: "Expected property name or '}' in JSON at position 1" → 第 1 行 第 2 列
    expect(r.line).toBe(1)
    expect(r.col).toBe(2)
  })
})

describe('jsonErrorPosition 位置提取', () => {
  it('Firefox "line N column M" 报文直接提取', () => {
    const pos = jsonErrorPosition(new Error('JSON.parse: bad at line 3 column 5 of the JSON data'), 'x')
    expect(pos).toEqual({ line: 3, col: 5 })
  })
  it('V8 position 报文按文本换算多行行:列', () => {
    const pos = jsonErrorPosition(new Error('Unexpected non-whitespace character after JSON at position 8'), '{\n"a":1\n}')
    expect(pos).toEqual({ line: 3, col: 1 })
  })
  it('position 超出文本长度 / 无定位报文 → 空对象', () => {
    expect(jsonErrorPosition(new Error('after JSON at position 99'), '{a}')).toEqual({})
    expect(jsonErrorPosition(new Error('no position info'), '{a}')).toEqual({})
  })
  it('源码摘录型报文(无 position/line)→ 空对象,调用方降级', () => {
    const excerpt = ['Unexpected token in JSON: {', '  "a": bad}'].join('')
    expect(jsonErrorPosition(new Error(excerpt), '{\n  "a": bad\n}')).toEqual({})
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
