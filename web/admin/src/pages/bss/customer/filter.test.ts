import { describe, expect, it } from 'vitest'
import { filterCustomers, pageSlice } from './filter'
import type { CustomerRow } from './types'

const row = (over: Partial<CustomerRow>): CustomerRow => ({
  id: 1, name: 'Juan', phone: '09171234567', idType: '身份证', idNo: 'ID9001',
  realNameStatus: 'VERIFIED', serviceStatus: 'ACTIVE', addressId: 1, legalEntityId: 1,
  regionId: 1, regionName: 'Metro Manila', createdAt: '2026-01-01T00:00:00Z', ...over,
})

describe('filterCustomers', () => {
  const rows = [row({}), row({ name: 'Maria Santos', idNo: 'ID9002', phone: '09189998887' })]

  it('keyword 匹配姓名(不分大小写)', () => {
    expect(filterCustomers(rows, 'juan', '')).toHaveLength(1)
    expect(filterCustomers(rows, 'maria', '')[0].idNo).toBe('ID9002')
  })
  it('keyword 匹配证件号', () => {
    expect(filterCustomers(rows, 'id9002', '')).toHaveLength(1)
  })
  it('phone 前缀/包含匹配', () => {
    expect(filterCustomers(rows, '', '0918')).toHaveLength(1)
    expect(filterCustomers(rows, '', '0917')).toHaveLength(1)
    expect(filterCustomers(rows, '', '0000')).toHaveLength(0)
  })
  it('空条件全量', () => {
    expect(filterCustomers(rows, ' ', ' ')).toHaveLength(2)
  })
})

describe('pageSlice', () => {
  it('按页切片', () => {
    expect(pageSlice([1, 2, 3, 4, 5], 2, 2)).toEqual([3, 4])
    expect(pageSlice([1], 3, 2)).toEqual([])
  })
})
