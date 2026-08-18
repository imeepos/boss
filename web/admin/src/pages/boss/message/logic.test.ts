import { describe, expect, it } from 'vitest'
import { filterMessages, filterNotices, fmtTime, toId, type NoticeEntry, type WorkerMessageEntry } from './logic'

const msg = (over: Partial<WorkerMessageEntry>): WorkerMessageEntry => ({
  id: 1, workerId: 7, level: 'INFO', title: '改派通知', content: 'TKT-001 已改派', sentAt: '2025-08-17T08:00:00Z', read: false, ...over,
})

describe('filterMessages', () => {
  const rows = [
    msg({ id: 1, level: 'URGENT', title: '台风预警', content: '注意安全', read: false }),
    msg({ id: 2, level: 'WARN', title: '超时提醒', content: '工单即将超时', read: true }),
    msg({ id: 3, workerId: 9, title: '佣金到账', content: '本月佣金', read: true }),
  ]
  it('关键字命中标题或内容(不区分大小写)', () => {
    expect(filterMessages(rows, '台风', '', '').map((r) => r.id)).toEqual([1])
    expect(filterMessages(rows, '超时', '', '').map((r) => r.id)).toEqual([2])
  })
  it('级别与已读状态组合过滤', () => {
    expect(filterMessages(rows, '', 'URGENT', '').map((r) => r.id)).toEqual([1])
    expect(filterMessages(rows, '', '', 'unread').map((r) => r.id)).toEqual([1])
    expect(filterMessages(rows, '', 'WARN', 'read').map((r) => r.id)).toEqual([2])
    expect(filterMessages(rows, '', 'INFO', 'unread')).toEqual([])
  })
})

describe('filterNotices', () => {
  const rows: NoticeEntry[] = [
    { id: 1, title: '系统维护', category: 'ops', active: true, publishedAt: '2025-08-01T00:00:00Z' },
    { id: 2, title: '活动下线', category: 'promo', active: false, publishedAt: '2025-08-02T00:00:00Z' },
  ]
  it('activeOnly 只保留上架', () => {
    expect(filterNotices(rows, '', true).map((r) => r.id)).toEqual([1])
  })
  it('关键字命中标题/分类', () => {
    expect(filterNotices(rows, 'promo', false).map((r) => r.id)).toEqual([2])
    expect(filterNotices(rows, '', false)).toHaveLength(2)
  })
})

describe('fmtTime/toId', () => {
  it('ISO 时间转本地短格式,空值兜底 -', () => {
    expect(fmtTime('')).toBe('-')
    expect(fmtTime('abc')).toBe('abc')
    expect(fmtTime('2025-08-17T08:00:00Z')).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/)
  })
  it('师傅ID容错解析', () => {
    expect(toId('42')).toBe(42)
    expect(toId(' 7 ')).toBe(7)
    expect(toId('abc')).toBe(0)
    expect(toId('-3')).toBe(0)
  })
})
