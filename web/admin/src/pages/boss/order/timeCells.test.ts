// T16 口径测试:① 列表「下单时间」列统一走 fmtTime(上海墙钟);② 时间轴 finishedAt
// 缺失显示 i18n 状态文案(如「未完成」)而非裸 —;③ 三语列名/降级文案键齐备且位置一致。
import { describe, expect, it } from 'vitest'
import zhCN from '../../../i18n/locales/zh-CN'
import enUS from '../../../i18n/locales/en-US'
import msMY from '../../../i18n/locales/ms-MY'
import { createdAtText, timelineFinishedText } from './timeCells'

describe('createdAtText 订单列表下单时间', () => {
  it('RFC3339 → 上海墙钟 YYYY-MM-DD HH:mm:ss(复用 fmtTime)', () => {
    expect(createdAtText('2026-09-03T16:30:00Z')).toBe('2026-09-04 00:30:00')
    expect(createdAtText('2026-09-03T08:30:00+08:00')).toBe('2026-09-03 08:30:00')
    expect(createdAtText('2026-09-02T19:30:00-05:00')).toBe('2026-09-03 08:30:00')
  })
  it('空值降级为 —(与 fmtTime 空值口径一致)', () => {
    expect(createdAtText(null)).toBe('—')
    expect(createdAtText(undefined)).toBe('—')
    expect(createdAtText('')).toBe('—')
  })
})

describe('timelineFinishedText 时间轴完成时间缺失降级', () => {
  it('有值时走 fmtTime 上海墙钟', () => {
    expect(timelineFinishedText('2026-09-02T19:30:00-05:00', '未完成')).toBe('2026-09-03 08:30:00')
  })
  it('缺失(null/undefined/空串)显示 i18n 状态文案,不显示裸 —', () => {
    expect(timelineFinishedText(null, '未完成')).toBe('未完成')
    expect(timelineFinishedText(undefined, 'Not finished')).toBe('Not finished')
    expect(timelineFinishedText('', 'Belum selesai')).toBe('Belum selesai')
  })
})

describe('三语列名与降级文案', () => {
  it.each([
    ['zh-CN', zhCN, '下单时间'],
    ['en-US', enUS, 'Order Time'],
    ['ms-MY', msMY, 'Masa Pesanan'],
  ] as const)('%s:下单时间列位于状态与操作之间', (_name, dict, label) => {
    const cols = dict.pages.orderPage.columns
    expect(cols[cols.length - 2]).toBe(label)
    expect(cols[cols.length - 1]).not.toBe(label)
  })
  it('三语 timelineUnfinished 非空且 zh-CN 为「未完成」', () => {
    expect(zhCN.pages.orderPage.timelineUnfinished).toBe('未完成')
    for (const dict of [zhCN, enUS, msMY]) {
      expect(dict.pages.orderPage.timelineUnfinished.length).toBeGreaterThan(0)
    }
  })
})
