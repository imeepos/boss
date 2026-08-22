// stacked-bars 单测:max 归一 + 堆叠比例 + 图例渲染。
import { describe, expect, it } from 'vitest'
import { StackedBars } from './stacked-bars'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

describe('StackedBars', () => {
  it('空数组返回 null(无 DOM)', () => {
    const html = renderToStaticMarkup(createElement(StackedBars, { groups: [], legends: [] }))
    expect(html).toBe('')
  })
  it('max 归一高度(最大柱 100%)', () => {
    const html = renderToStaticMarkup(
      createElement(StackedBars, {
        groups: [{ stacks: [1, 1] }, { stacks: [4, 4] }],
        legends: ['A', 'B'],
      }),
    )
    expect(html).toContain('height:50%')
    expect(html).toContain('height:100%')
  })
  it('图例渲染 K 项', () => {
    const html = renderToStaticMarkup(
      createElement(StackedBars, {
        groups: [{ stacks: [1, 2, 3] }],
        legends: ['A', 'B', 'C'],
      }),
    )
    expect(html).toContain('>A<')
    expect(html).toContain('>B<')
    expect(html).toContain('>C<')
  })
})