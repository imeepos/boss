// donut 单测:sum 兜底 + 总数覆盖 + 百分比之和 = 100% + 格式化回调。
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

  it('label/sublabel props 渲染', () => {
    const html = renderToStaticMarkup(
      createElement(Donut, {
        segments: [{ label: 'A', value: 1 }, { label: 'B', value: 1 }],
        label: '指标构成',
        sublabel: '5 项',
      }),
    )
    expect(html).toContain('>指标构成<')
    expect(html).toContain('>5 项<')
  })

  it('formatTotal 回调覆盖默认千分位', () => {
    const html = renderToStaticMarkup(
      createElement(Donut, {
        segments: [{ label: 'A', value: 1 }, { label: 'B', value: 1 }],
        formatTotal: (v) => `T-${v}`,
      }),
    )
    expect(html).toContain('>T-2<')
  })

  it('formatValue 回调格式化 legend 数值(替代原始浮点)', () => {
    // 不传 formatValue:仍显示原始浮点
    const htmlRaw = renderToStaticMarkup(
      createElement(Donut, { segments: [{ label: 'A', value: 0.5076923076923077 }] }),
    )
    expect(htmlRaw).toContain('0.5076923')
    // 传 formatValue:替换为格式化输出(签名接 v + segment)
    const htmlFmt = renderToStaticMarkup(
      createElement(Donut, {
        segments: [{ label: '端口利用率', value: 0.5076923076923077 }],
        formatValue: (v) => `${(v * 100).toFixed(1)}%`,
      }),
    )
    expect(htmlFmt).toContain('· 50.8%')
    expect(htmlFmt).not.toContain('0.5076923')
  })
})