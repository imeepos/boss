import { describe, expect, it } from 'vitest'
import { filterTables, formatBytes, formatTime, isBusyError, toJob } from './logic'

describe('formatBytes', () => {
  it('0 与负数 → 0 B', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(-5)).toBe('0 B')
  })
  it('逐级进位', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(2048)).toBe('2.0 KB')
    expect(formatBytes(1536 * 1024)).toBe('1.5 MB')
    expect(formatBytes(5 * 1024 * 1024 * 1024)).toBe('5.0 GB')
  })
})

describe('formatTime', () => {
  it('空串透传', () => {
    expect(formatTime('')).toBe('')
  })
  it('非法输入原样返回', () => {
    expect(formatTime('not-a-date')).toBe('not-a-date')
  })
  it('合法 RFC3339 格式化为 YYYY-MM-DD HH:mm', () => {
    const out = formatTime('2026-08-21T09:05:00Z')
    expect(out).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/)
  })
})

describe('filterTables', () => {
  it('空关键词返回全集', () => {
    expect(filterTables(['accounts', 'orders'], '  ')).toEqual(['accounts', 'orders'])
  })
  it('大小写不敏感子串', () => {
    expect(filterTables(['accounts', 'geo_country', 'orders'], 'COUNTRY')).toEqual(['geo_country'])
  })
})

describe('isBusyError', () => {
  it('42300 → true', () => {
    expect(isBusyError({ code: 42300 })).toBe(true)
  })
  it('其他 code/非对象 → false', () => {
    expect(isBusyError({ code: 50000 })).toBe(false)
    expect(isBusyError(new Error('x'))).toBe(false)
  })
})

describe('toJob', () => {
  it('容错映射缺省字段', () => {
    const j = toJob({
      id: 3, kind: 'backup', scope: 'all', tables: null as unknown as string[], status: 'running',
      fileName: '', sizeBytes: '12' as unknown as number, tableCount: 0, rowCount: 0,
      operator: '', createdAt: '',
    })
    expect(j.tables).toEqual([])
    expect(j.sizeBytes).toBe(12)
  })
})
