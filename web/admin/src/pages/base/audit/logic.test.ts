import { describe, expect, it } from 'vitest'
import { formatTime } from './logic'

// 回归(ISSUE.md 展示错位):formatTime 曾字符串截断露出 UTC 墙钟,
// 菲律宾用户看到慢 8 小时;必须经 Date 转浏览器本地时区。
describe('audit formatTime', () => {
  it('UTC ISO → 本地时区墙钟(与 Date 本地格式化一致)', () => {
    const iso = '2026-08-21T23:52:48Z'
    const d = new Date(iso)
    const p = (x: number) => String(x).padStart(2, '0')
    const want = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
    expect(formatTime(iso)).toBe(want)
    expect(formatTime(iso)).not.toBe('2026-08-21 23:52:48') // 除非机器恰在 UTC
  })
  it('空值/非法值兜底', () => {
    expect(formatTime('')).toBe('')
    expect(formatTime('not-a-date')).toBe('not-a-date')
  })
})
