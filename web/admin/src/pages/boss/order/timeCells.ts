// 订单域时间单元格口径(T16):列表「下单时间」与时间轴「完成时间」的展示规则集中在此,便于单测。
// 统一复用 lib/format fmtTime(上海墙钟 YYYY-MM-DD HH:mm:ss);缺失值给显式状态文案,不显示裸 —。
import { fmtTime } from '../../../lib/format'

/** 订单列表「下单时间」:createdAt → fmtTime;空/非法沿用 fmtTime 口径(空 → —)。 */
export function createdAtText(createdAt: string | undefined | null): string {
  return fmtTime(createdAt)
}

/** 时间轴「完成时间」:finishedAt 缺失(环节未完成/未回填)显示 i18n 状态文案(如「未完成」)。 */
export function timelineFinishedText(finishedAt: string | undefined | null, unfinished: string): string {
  return finishedAt ? fmtTime(finishedAt) : unfinished
}
