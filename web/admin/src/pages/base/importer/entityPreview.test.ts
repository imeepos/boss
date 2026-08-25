// 业务实体批量导入纯函数单测:行矫正/校验/模板 + Excel 通道回环。
import { describe, expect, it } from 'vitest'
import * as XLSX from 'xlsx'
import { findEntity, IMPORT_ENTITIES } from './entities'
import { coerceEntityRow, entityTemplateJson, parseEntityRows } from './entityPreview'
import { entityExcelTemplate, parseEntityExcel } from './excel'

describe('实体定义', () => {
  it('5 个项目均含端点与必填列', () => {
    expect(IMPORT_ENTITIES.map((e) => e.kind)).toEqual(['account', 'legalEntity', 'department', 'post', 'product'])
    for (const e of IMPORT_ENTITIES) {
      expect(e.endpoint.startsWith('/')).toBe(true)
      expect(e.columns.some((c) => c.required)).toBe(true)
      expect(e.samples.length).toBeGreaterThan(0)
    }
  })
})

describe('coerceEntityRow', () => {
  const product = findEntity('product') as NonNullable<ReturnType<typeof findEntity>>
  it('number 转数值、可选空串剔除、字符串 trim', () => {
    const r = coerceEntityRow(product, { legalEntityId: ' 1 ', name: ' 套餐 ', monthlyFee: '99.5', bandwidth: '', status: 'DRAFT' })
    expect(r).toEqual({ ok: true, body: { legalEntityId: 1, name: '套餐', monthlyFee: 99.5, status: 'DRAFT' } })
  })
  it('必填缺失/数值非法返回字段名', () => {
    expect(coerceEntityRow(product, { legalEntityId: 1, name: 'x' })).toEqual({ ok: false, field: 'monthlyFee' })
    expect(coerceEntityRow(product, { legalEntityId: 1, name: 'x', monthlyFee: 'abc' })).toEqual({ ok: false, field: 'monthlyFee' })
  })
  it('list 列:字符串按中英逗号/分号拆分,数组直通,空值剔除', () => {
    const post = findEntity('post') as NonNullable<ReturnType<typeof findEntity>>
    const base = { deptId: 1, code: 'P1', name: '岗' }
    expect((coerceEntityRow(post, { ...base, roles: 'a，b；c' }) as { body: Record<string, unknown> }).body.roles)
      .toEqual(['a', 'b', 'c'])
    expect((coerceEntityRow(post, { ...base, roles: ['x', ' y '] }) as { body: Record<string, unknown> }).body.roles)
      .toEqual(['x', 'y'])
    expect((coerceEntityRow(post, { ...base, roles: '' }) as { body: Record<string, unknown> }).body.roles)
      .toBeUndefined()
  })
})

describe('parseEntityRows', () => {
  const dept = findEntity('department') as NonNullable<ReturnType<typeof findEntity>>
  it('接受数组与 {rows} 信封,数值列转数值', () => {
    expect(parseEntityRows(dept, [{ legalEntityId: 1, name: 'A' }, { legalEntityId: '2', name: 'B' }]))
      .toEqual({ ok: true, rows: [{ legalEntityId: 1, name: 'A' }, { legalEntityId: 2, name: 'B' }] })
    expect(parseEntityRows(dept, { rows: [{ legalEntityId: 1, name: 'A' }] }))
      .toEqual({ ok: true, rows: [{ legalEntityId: 1, name: 'A' }] })
  })
  it('非数组/坏行定位行号与字段', () => {
    expect(parseEntityRows(dept, { x: 1 })).toEqual({ ok: false, reason: 'notArray' })
    expect(parseEntityRows(dept, [{ legalEntityId: 1, name: 'A' }, { legalEntityId: 1 }]))
      .toEqual({ ok: false, reason: 'badRow', row: 2, field: 'name' })
  })
})

describe('模板与 Excel 回环', () => {
  it('JSON 模板可被 parseEntityRows 接受', () => {
    for (const e of IMPORT_ENTITIES) {
      expect(parseEntityRows(e, JSON.parse(entityTemplateJson(e))).ok).toBe(true)
    }
  })
  it('Excel 模板回环解析为合法行', () => {
    for (const e of IMPORT_ENTITIES) {
      const r = parseEntityExcel(e, entityExcelTemplate(e))
      expect(r.ok).toBe(true)
      if (r.ok) expect(parseEntityRows(e, r.value).ok).toBe(true)
    }
  })
  it('表头缺列报 badHeader,空数据报 badRow', () => {
    const dept = findEntity('department') as NonNullable<ReturnType<typeof findEntity>>
    const wb = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet([['legalEntityId'], [1]]), dept.kind)
    const badHeader = parseEntityExcel(dept, XLSX.write(wb, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer)
    expect(badHeader).toEqual({ ok: false, reason: 'badHeader', sheet: dept.kind })
    const wb2 = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb2, XLSX.utils.aoa_to_sheet([['legalEntityId', 'name'], ['', '']]), dept.kind)
    const noData = parseEntityExcel(dept, XLSX.write(wb2, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer)
    expect(noData.ok).toBe(false)
  })
})
