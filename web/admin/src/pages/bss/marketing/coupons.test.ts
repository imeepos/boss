// 券表单数值解析回归:空串=0,非法/负数/小数计数拒绝。
import { describe, it, expect } from 'vitest'
import { toCents, toCount } from './coupons'

describe('toCents', () => {
  it('空串与 0 都返回 0 分', () => {
    expect(toCents('')).toBe(0)
    expect(toCents('  ')).toBe(0)
    expect(toCents('0')).toBe(0)
  })
  it('正常金额四舍五入到分', () => {
    expect(toCents('50')).toBe(5000)
    expect(toCents('12.345')).toBe(1235)
  })
  it('负数与非数字拒绝', () => {
    expect(toCents('-1')).toBeNull()
    expect(toCents('abc')).toBeNull()
  })
})

describe('toCount', () => {
  it('空串=0,整数放行', () => {
    expect(toCount('')).toBe(0)
    expect(toCount('30')).toBe(30)
  })
  it('小数/负数/非数字拒绝', () => {
    expect(toCount('1.5')).toBeNull()
    expect(toCount('-2')).toBeNull()
    expect(toCount('x')).toBeNull()
  })
})