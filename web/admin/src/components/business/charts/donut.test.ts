// donut 单测:sum 兜底 + 总数覆盖 + 百分比之和 = 100%。
// 不依赖 @testing-library(项目 vitest 仅纯函数/逻辑测试);走边界断言。
import { describe, expect, it } from 'vitest'
import { Donut } from './donut'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

describe('Donut', () => {
  it('空数据兜底显示 —', () => {
    const html = renderToStaticMarkup(createElement(Donut, { segments: [] }))
    expect(html).toContain('—')
  })
  it('总和走显式 total', () => {
    const html = renderToStaticMarkup(
      createElement(Donut, { total: 200, segments: [
        { label: 'A', value: 80 }, { label: 'B', value: 120 },
      ] }),
    )
    expect(html).toContain('>200<')
  })
  it('渲染 N 段 circle', () => {
    const html = renderToStaticMarkup(
      createElement(Donut, { segments: [
        { label: 'A', value: 1 }, { label: 'B', value: 1 }, { label: 'C', value: 1 },
      ] }),
    )
    expect((html.match(/<circle/g) ?? []).length).toBe(3)
  })
  it('百分比之和 = 100%', () => {
    const html = renderToStaticMarkup(
      createElement(Donut, { segments: [
        { label: 'A', value: 3 }, { label: 'B', value: 1 },
      ] }),
    )
    expect(html).toContain('75.0%')
    expect(html).toContain('25.0%')
  })
})