// v-bars 单测:空态文案 + valueLabel 后缀 + 高度归一。
import { describe, expect, it } from 'vitest'
import { VerticalBars } from './v-bars'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

describe('VerticalBars', () => {
  it('空 labels → 显示默认 emptyText"暂无数据"', () => {
    const html = renderToStaticMarkup(createElement(VerticalBars, { labels: [], values: [] }))
    expect(html).toContain('暂无数据')
  })
  it('空 values + 自定义 emptyText', () => {
    const html = renderToStaticMarkup(
      createElement(VerticalBars, { labels: [], values: [], emptyText: '尚无地址数据' }),
    )
    expect(html).toContain('尚无地址数据')
  })
  it('valueLabel 加柱顶数字后缀', () => {
    const html = renderToStaticMarkup(
      createElement(VerticalBars, {
        labels: ['A', 'B'],
        values: [50, 80],
        valueLabel: '%',
      }),
    )
    expect(html).toContain('>50%<')
    expect(html).toContain('>80%<')
    expect(html).toContain('A: 50%')  // title 属性
  })
  it('无 valueLabel 时显示原始数字', () => {
    const html = renderToStaticMarkup(
      createElement(VerticalBars, { labels: ['A'], values: [42] }),
    )
    expect(html).toContain('>42<')
    expect(html).not.toContain('42%')
  })
  it('max 归一:最大值为 100% 高', () => {
    const html = renderToStaticMarkup(
      createElement(VerticalBars, { labels: ['A', 'B'], values: [50, 100] }),
    )
    // 第二个柱 height: 100%,第一个 height: 50%
    expect(html).toContain('height:50%')
    expect(html).toContain('height:100%')
  })
})