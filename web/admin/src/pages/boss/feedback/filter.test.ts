// 回访评价筛选逻辑回归:师傅关键字(不区分大小写、trim)+ 只看待复核。
// 抽出 filterFeedback 以便纯函数测试(vitest environment=node)。
import { describe, it, expect } from 'vitest'
import type { FeedbackRow } from '../types'
import { filterFeedback } from './filter'

const sample: FeedbackRow[] = [
  { id: 11, workerId: 1, workerName: '张伟', groupId: 1, groupName: '东区一组', legalEntityId: 1, legalEntityName: '上海电信', regionId: 1, regionName: '浦东', ticketId: 88001, customerId: 1, customerName: '王女士', score: 2, needReview: true },
  { id: 12, workerId: 2, workerName: '陈强', groupId: 1, groupName: '东区一组', legalEntityId: 1, legalEntityName: '上海电信', regionId: 1, regionName: '浦东', ticketId: 88002, customerId: 2, customerName: '李先生', score: 5, needReview: false },
  { id: 13, workerId: 3, workerName: '刘洋', groupId: 2, groupName: '南区二组', legalEntityId: 1, legalEntityName: '上海电信', regionId: 2, regionName: '徐汇', ticketId: 88003, customerId: 3, customerName: '赵女士', score: 1, needReview: true },
]

describe('filterFeedback', () => {
  it('空筛选条件返回全部', () => {
    expect(filterFeedback(sample, { workerKw: '', reviewOnly: false })).toEqual(sample)
  })

  it('worker 关键字匹配名字子串(不区分大小写 + trim)', () => {
    expect(filterFeedback(sample, { workerKw: '  陈强  ', reviewOnly: false }).map((r) => r.id)).toEqual([12])
    expect(filterFeedback(sample, { workerKw: 'zhang', reviewOnly: false })).toEqual([])
    // 区分语言:中文匹配应走精确包含(此例 "伟" 出现在"张伟")
    expect(filterFeedback(sample, { workerKw: '伟', reviewOnly: false }).map((r) => r.id)).toEqual([11])
  })

  it('reviewOnly 仅保留 needReview=true 行', () => {
    expect(filterFeedback(sample, { workerKw: '', reviewOnly: true }).map((r) => r.id)).toEqual([11, 13])
  })

  it('reviewOnly + worker 关键字组合收敛', () => {
    expect(filterFeedback(sample, { workerKw: '刘洋', reviewOnly: true }).map((r) => r.id)).toEqual([13])
    expect(filterFeedback(sample, { workerKw: '陈强', reviewOnly: true })).toEqual([])
  })

  it('空数据不抛', () => {
    expect(filterFeedback([], { workerKw: 'a', reviewOnly: true })).toEqual([])
  })
})