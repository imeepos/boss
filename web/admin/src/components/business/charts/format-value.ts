// 指标值按 key 启发式格式化:后端 indicator.key 命名规范,无需契约扩展。
// F1:把"427.4724358974359" → "427.47",0.508 → "50.8%" 等。
//
// Key 派生映射(与 internal/domain/analytics/pg.go:67-72 对齐):
//   portUtilization / installConversion → 0~1 比例 → 百分比
//   maintenanceCostPerUser              → 金额
//   assetHealth                         → 0~100 评分(取整)
//   regionROI                           → 比值(2 位小数)
//   其它                                → 启发式:0~1 走百分比,绝对数走千分位 2 位小数

import type { IndicatorRow } from '../../../pages/intel/types'

/** 千分位整数(12345.67 → "12,345.67")。NaN/Infinity 兜底为 "0"。 */
export function formatNumber(n: number, decimals = 2): string {
  if (!Number.isFinite(n)) return '0'
  return n.toLocaleString('en-US', {
    minimumFractionDigits: 0,
    maximumFractionDigits: decimals,
  })
}

/** 0~1 比例 → 百分比("50.8%")。值>1 时按原值百分比("150%")。NaN 兜底 "0%"。 */
export function formatRatio(n: number, decimals = 1): string {
  if (!Number.isFinite(n)) return '0%'
  return `${(n * 100).toFixed(decimals)}%`
}

/** 金额使用千分位并在过大时切换为万/亿单位。 */
export function formatCurrency(n: number): string {
  if (!Number.isFinite(n)) return '¥0'
  const absolute = Math.abs(n)
  if (absolute >= 100_000_000) return `¥${(n / 100_000_000).toFixed(2)}亿`
  if (absolute >= 10_000) return `¥${(n / 10_000).toFixed(2)}万`
  return `¥${formatNumber(n, 2)}`
}

/** 将后端 ROI 详情中的金额也压缩，避免明细抽屉出现长数字。 */
export function formatIndicatorDetail(detail: string): string {
  return detail.replace(/(收入|投资)(-?\d+(?:\.\d+)?)(元)/g, (_, label, value) => `${label}${formatCurrency(Number(value))}`)
}

/** 评分(0~100,取整)。 */
export function formatScore(n: number): string {
  if (!Number.isFinite(n)) return '0'
  return Math.round(n).toString()
}

/** 按 indicator.key 派格式(主入口:Donut/analyze 页都走这里)。 */
export function formatIndicatorValue(key: string, value: number): string {
  if (key === 'portUtilization' || key === 'installConversion') return formatRatio(value, 1)
  if (key === 'maintenanceCostPerUser') return formatCurrency(value)
  if (key === 'assetHealth') return formatScore(value)
  if (key === 'regionROI') return formatNumber(value, 2)
  if (Number.isFinite(value) && value > 0 && value <= 1) return formatRatio(value, 1)
  return formatNumber(value, 2)
}

/** 简化的 Donut 显示形态(用 value 而非 sum,因 value 已是单项值)。 */
export function formatSegmentValue(ind: IndicatorRow): string {
  return formatIndicatorValue(ind.key, ind.value)
}
