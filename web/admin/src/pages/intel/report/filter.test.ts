// periodLabel / filterReports 纯函数回归:周期可读名、关键词命中(ID/原始键/本地化名)。
import { describe, it, expect } from 'vitest'
import { PERIODS, periodLabel, filterReports, type PeriodLabels } from './filter'
import type { ReportRow } from '../types'

const labels: PeriodLabels = {
  periods: ['日报', '周报', '月报', '季报'],
  periodRecon: '对账日报',
  periodOssAudit: '稽核日报',
}

function row(id: number, period: string): ReportRow {
  return { id, period, windowStart: '', windowEnd: '', createdAt: '' }
}

const rows = [row(1, 'daily'), row(2, 'recon-daily'), row(3, 'oss-audit-daily'), row(4, 'weekly')]

describe('periodLabel', () => {
  it('标准周期查文案表', () => {
    expect(PERIODS.map((p) => periodLabel(p, labels))).toEqual(['日报', '周报', '月报', '季报'])
  })
  it('特殊流水线快照走专名', () => {
    expect(periodLabel('recon-daily', labels)).toBe('对账日报')
    expect(periodLabel('oss-audit-daily', labels)).toBe('稽核日报')
  })
  it('未知周期原样透出不吞数据', () => {
    expect(periodLabel('custom-x', labels)).toBe('custom-x')
  })
})

describe('filterReports', () => {
  it('period 等值过滤;空串=全部', () => {
    expect(filterReports(rows, 'daily', '', labels).map((x) => x.id)).toEqual([1])
    expect(filterReports(rows, '', '', labels)).toHaveLength(4)
  })
  it('q 命中 ID', () => {
    expect(filterReports(rows, '', '3', labels).map((x) => x.id)).toEqual([3])
  })
  it('q 命中原始周期键(大小写不敏感)', () => {
    expect(filterReports(rows, '', 'RECON', labels).map((x) => x.id)).toEqual([2])
    expect(filterReports(rows, '', 'daily', labels)).toHaveLength(3)
  })
  it('q 命中本地化周期名', () => {
    expect(filterReports(rows, '', '对账', labels).map((x) => x.id)).toEqual([2])
    expect(filterReports(rows, '', '稽核', labels).map((x) => x.id)).toEqual([3])
    expect(filterReports(rows, '', '周报', labels).map((x) => x.id)).toEqual([4])
  })
  it('q 首尾空白忽略;无命中返回空', () => {
    expect(filterReports(rows, '', ' 对账 ', labels)).toHaveLength(1)
    expect(filterReports(rows, '', '不存在', labels)).toHaveLength(0)
  })
  it('period 与 q 组合过滤', () => {
    expect(filterReports(rows, 'recon-daily', '2', labels).map((x) => x.id)).toEqual([2])
    expect(filterReports(rows, 'daily', '对账', labels)).toHaveLength(0)
  })
})