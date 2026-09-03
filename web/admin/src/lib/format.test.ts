import { describe, expect, it } from 'vitest'
import { bizDateKey, fmtFee, fmtTime } from './format'

describe('fmtFee', () => {
  it('keeps ordinary amounts precise and readable', () => {
    expect(fmtFee(1234.5)).toBe('¥1,234.50')
  })

  it('uses compact units for large amounts', () => {
    expect(fmtFee(123456)).toBe('¥12.35万')
    expect(fmtFee(123456789)).toBe('¥1.23亿')
  })

  it('handles negative and invalid values', () => {
    expect(fmtFee(-123456789)).toBe('¥-1.23亿')
    expect(fmtFee(Number.NaN)).toBe('¥0')
  })
})

describe('fmtTime', () => {
  it('renders UTC instants on the Asia/Shanghai wall clock', () => {
    expect(fmtTime('2026-09-03T16:30:00Z')).toBe('2026-09-04 00:30:00')
  })

  it('normalizes offset-bearing inputs to the same business clock', () => {
    expect(fmtTime('2026-09-03T08:30:00+08:00')).toBe('2026-09-03 08:30:00')
    expect(fmtTime('2026-09-02T19:30:00-05:00')).toBe('2026-09-03 08:30:00')
  })

  it('falls back on empty and invalid values', () => {
    expect(fmtTime(null)).toBe('—')
    expect(fmtTime(undefined)).toBe('—')
    expect(fmtTime('')).toBe('—')
    expect(fmtTime('not-a-timestamp')).toBe('not-a-timestamp')
  })
})

describe('bizDateKey', () => {
  it('keys the Shanghai calendar day for business-day filters', () => {
    expect(bizDateKey('2026-09-03T16:30:00Z')).toBe('2026-09-04')
    expect(bizDateKey('2026-09-03T15:59:59Z')).toBe('2026-09-03')
    expect(bizDateKey(new Date('2026-09-03T16:30:00Z'))).toBe('2026-09-04')
    expect(bizDateKey('garbage')).toBe('')
  })
})
