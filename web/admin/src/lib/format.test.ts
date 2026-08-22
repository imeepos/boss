import { describe, expect, it } from 'vitest'
import { fmtFee } from './format'

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
