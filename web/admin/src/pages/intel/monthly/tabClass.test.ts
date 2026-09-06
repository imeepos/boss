import { describe, expect, it } from 'vitest'
import { monthlyTabClass } from './tabClass'

const utility = (cls: string, prefix: string): string[] =>
  cls.split(' ').filter((c) => c.startsWith(prefix))

// 回归:激活/空闲页签的 bg-*/text-* 工具类曾同时挂在同一元素,
// 生效方由 CSS 产物顺序裁决——亮色主题白字白底、暗色主题深底深字。
describe('monthlyTabClass 双主题互斥回归', () => {
  const active = monthlyTabClass(true)
  const idle = monthlyTabClass(false)

  it('激活态用 fab 强调令牌(亮藏青/暗金随主题)', () => {
    expect(utility(active, 'bg-')).toEqual(['bg-[var(--shell-fab-bg)]'])
    expect(utility(active, 'text-[var(')).toEqual(['text-[var(--shell-fab-icon)]'])
  })

  it('空闲态用输入/内容令牌', () => {
    expect(utility(idle, 'bg-')).toEqual(['bg-[var(--shell-input-bg)]'])
    expect(utility(idle, 'text-[var(')).toEqual(['text-[var(--shell-content-text)]'])
  })

  it('active/idle 状态类互斥,无任何重叠工具类', () => {
    for (const prefix of ['bg-', 'text-[var(', 'border-[var(', 'hover:text-']) {
      const overlap = utility(active, prefix).filter((c) => utility(idle, prefix).includes(c))
      expect(overlap).toEqual([])
    }
  })
})
