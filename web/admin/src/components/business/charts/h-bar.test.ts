// h-bar 单测:render + max 归一 + format。
import { describe, expect, it } from 'vitest'
import { HorizontalBar } from './h-bar'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

describe('HorizontalBar', () => {
  it('按 max 归一宽度', () => {
    const html = renderToStaticMarkup(
      createElement(HorizontalBar, { items: [
        { label: 'A', value: 50 }, { label: 'B', value: 100 },
      ] }),
    )
    expect(html).toContain('width:50%')
    expect(html).toContain('width:100%')
  })
  it('format 回调', () => {
    const html = renderToStaticMarkup(
      createElement(HorizontalBar, {
        items: [{ label: 'X', value: 12.5 }],
        format: (v) => `${v}%`,
      }),
    )
    expect(html).toContain('12.5%')
  })
  it('空数组不崩', () => {
    const html = renderToStaticMarkup(createElement(HorizontalBar, { items: [] }))
    expect(html).toContain('<ul')
  })
})