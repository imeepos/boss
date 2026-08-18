import { describe, expect, it } from 'vitest'
import { filterLegalEntities, pageSlice, type LegalEntityRow } from './filter'

const rows: LegalEntityRow[] = [
  { id: 1, code: 'LEG-A', name: '甲公司' },
  { id: 2, code: 'LEG-B', name: '乙公司' },
]

describe('filterLegalEntities', () => {
  it('空关键词返回全量', () => {
    expect(filterLegalEntities(rows, '  ')).toHaveLength(2)
  })
  it('按编码命中(大小写不敏感)', () => {
    expect(filterLegalEntities(rows, 'leg-a')).toEqual([rows[0]])
  })
  it('按名称命中', () => {
    expect(filterLegalEntities(rows, '乙')).toEqual([rows[1]])
  })
})

describe('pageSlice', () => {
  it('切出当前页', () => {
    expect(pageSlice([1, 2, 3, 4, 5], 2, 2)).toEqual([3, 4])
  })
  it('越界页返回空', () => {
    expect(pageSlice([1], 9, 10)).toEqual([])
  })
})
