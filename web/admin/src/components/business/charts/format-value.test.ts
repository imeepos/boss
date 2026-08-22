// format-value 单测:覆盖五条 indicator 派格式 + 千分位/小数/NaN 边界。
import { describe, expect, it } from 'vitest'
import {
  formatNumber, formatRatio, formatCurrency, formatScore,
  formatIndicatorValue, formatSegmentValue,
} from './format-value'

describe('format-value', () => {
  describe('formatNumber', () => {
    it('千分位 + 2 位小数', () => {
      expect(formatNumber(427.4724)).toBe('427.47')
      expect(formatNumber(12345.6789)).toBe('12,345.68')
    })
    it('整数无小数点', () => {
      expect(formatNumber(1000)).toBe('1,000')
    })
    it('NaN/Infinity 兜底 0', () => {
      expect(formatNumber(NaN)).toBe('0')
      expect(formatNumber(Infinity)).toBe('0')
    })
  })

  describe('formatRatio', () => {
    it('0~1 → 百分比', () => {
      expect(formatRatio(0.5076, 1)).toBe('50.8%')
      expect(formatRatio(0, 1)).toBe('0.0%')
    })
    it('>1 时按原值比例(150% 而非 15000%)', () => {
      expect(formatRatio(1.5, 1)).toBe('150.0%')
    })
    it('NaN 兜底 0%', () => {
      expect(formatRatio(NaN)).toBe('0%')
    })
  })

  describe('formatCurrency', () => {
    it('元(千分位 + 2 位小数 + ¥)', () => {
      expect(formatCurrency(366.6667)).toBe('¥366.67')
    })
  })

  describe('formatScore', () => {
    it('评分取整', () => {
      expect(formatScore(60.4)).toBe('60')
      expect(formatScore(60.6)).toBe('61')
    })
  })

  describe('formatIndicatorValue', () => {
    it('portUtilization/installConversion → 百分比', () => {
      expect(formatIndicatorValue('portUtilization', 0.5076)).toBe('50.8%')
      expect(formatIndicatorValue('installConversion', 0.298)).toBe('29.8%')
    })
    it('maintenanceCostPerUser → ¥', () => {
      expect(formatIndicatorValue('maintenanceCostPerUser', 366.67)).toBe('¥366.67')
    })
    it('assetHealth → 取整', () => {
      expect(formatIndicatorValue('assetHealth', 60.4)).toBe('60')
    })
    it('regionROI → 千分位 2 位小数', () => {
      expect(formatIndicatorValue('regionROI', 0)).toBe('0')
      expect(formatIndicatorValue('regionROI', 1.5)).toBe('1.5')
    })
    it('未知 key 启发式(0~1 走百分比,绝对数走千分位)', () => {
      expect(formatIndicatorValue('newCustomMetric', 0.42)).toBe('42.0%')
      expect(formatIndicatorValue('newCustomMetric', 100)).toBe('100')
    })
  })

  describe('formatSegmentValue(完整 indicator 入参)', () => {
    it('用 key 派格式', () => {
      expect(formatSegmentValue({ key: 'portUtilization', name: '端口利用率', value: 0.5, detail: '1/2' })).toBe('50.0%')
    })
  })
})