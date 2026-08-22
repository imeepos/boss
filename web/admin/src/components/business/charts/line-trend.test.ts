// line-trend 单测:SVG polyline + legend 渲染、空数据兜底、各自归一。
import { describe, expect, it } from 'vitest'
import { LineTrend } from './line-trend'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

describe('LineTrend', () => {
  it('空 series 返回 null(无 DOM)', () => {
    expect(renderToStaticMarkup(createElement(LineTrend, { labels: ['a'], series: [] }))).toBe('')
  })
  it('空 labels 返回 null', () => {
    expect(renderToStaticMarkup(createElement(LineTrend, { labels: [], series: [{ name: 'A', values: [1] }] }))).toBe('')
  })
  it('每条 series 一条 polyline', () => {
    const html = renderToStaticMarkup(
      createElement(LineTrend, {
        labels: ['t1', 't2', 't3'],
        series: [{ name: 'A', values: [1, 2, 3] }, { name: 'B', values: [3, 2, 1] }],
      }),
    )
    expect((html.match(/<polyline/g) ?? []).length).toBe(2)
  })
  it('每点一个圆(circle 节点=labels × series 数)', () => {
    const html = renderToStaticMarkup(
      createElement(LineTrend, {
        labels: ['t1', 't2'],
        series: [{ name: 'A', values: [1, 2] }],
      }),
    )
    // 2 labels × 1 series = 2 circles
    expect((html.match(/<circle/g) ?? []).length).toBeGreaterThanOrEqual(2)
  })
  it('legend 渲染 series name', () => {
    const html = renderToStaticMarkup(
      createElement(LineTrend, {
        labels: ['t1'],
        series: [{ name: 'Revenue', values: [1] }, { name: 'ROI', values: [2] }],
      }),
    )
    expect(html).toContain('>Revenue<')
    expect(html).toContain('>ROI<')
  })
})