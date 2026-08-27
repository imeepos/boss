// 回访评价筛选逻辑:师傅关键字 + 只看待复核。
// 从 FeedbackPage 抽出以便纯函数测试(vitest environment=node)。
import type { FeedbackRow } from '../types'

export interface FeedbackFilter {
  workerKw: string
  reviewOnly: boolean
}

export function filterFeedback(rows: FeedbackRow[], f: FeedbackFilter): FeedbackRow[] {
  const kw = f.workerKw.trim().toLowerCase()
  return rows.filter((r) => {
    if (f.reviewOnly && !r.needReview) return false
    if (!kw) return true
    return r.workerName.toLowerCase().includes(kw)
  })
}