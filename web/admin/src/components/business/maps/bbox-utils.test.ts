// bbox-utils 单测:parseBbox 边界 + formatBbox 回环 + 默认视域。
import { describe, expect, it } from 'vitest'
import { parseBbox, formatBbox, DEFAULT_VIEW } from './bbox-utils'

describe('bbox-utils', () => {
  describe('parseBbox', () => {
    it('空串返回 null', () => {
      expect(parseBbox('')).toBeNull()
      expect(parseBbox('   ')).toBeNull()
    })
    it('4 段合法数值', () => {
      expect(parseBbox('121,31,122,32')).toEqual({ minLng: 121, minLat: 31, maxLng: 122, maxLat: 32 })
    })
    it('允许空格', () => {
      expect(parseBbox(' 121 , 31 , 122 , 32 ')).toEqual({ minLng: 121, minLat: 31, maxLng: 122, maxLat: 32 })
    })
    it('段数错抛错', () => {
      expect(() => parseBbox('121,31,122')).toThrow(/4 段/)
    })
    it('非数值抛错', () => {
      expect(() => parseBbox('121,31,abc,32')).toThrow(/非数值/)
    })
    it('顺序错抛错', () => {
      expect(() => parseBbox('122,31,121,32')).toThrow(/顺序/)
    })
  })
  describe('formatBbox', () => {
    it('序列化回字符串', () => {
      expect(formatBbox({ minLng: 121, minLat: 31, maxLng: 122, maxLat: 32 })).toBe('121,31,122,32')
    })
    it('parse → format 恒等', () => {
      const s = '121.5,31.1,122.7,32.9'
      expect(formatBbox(parseBbox(s)!)).toBe(s)
    })
  })
  it('DEFAULT_VIEW 是马尼拉大区', () => {
    expect(DEFAULT_VIEW.center[0]).toBeCloseTo(121.0)
    expect(DEFAULT_VIEW.center[1]).toBeCloseTo(14.6)
    expect(DEFAULT_VIEW.zoom).toBe(10)
  })
})