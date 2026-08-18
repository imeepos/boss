import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { StatusTag } from './index'
import { REGISTRY } from './registry'

// 契约:全部枚举注册且渲染不出 unknown;颜色语义集中定义一次。
describe('StatusTag 注册表', () => {
  it('覆盖 16 个枚举域(与 server-ts/src/enums.ts 状态类对齐)', () => {
    expect(Object.keys(REGISTRY).length).toBeGreaterThanOrEqual(16)
  })

  it('每域非空且每值带 label/color', () => {
    for (const [domain, vals] of Object.entries(REGISTRY)) {
      expect(Object.keys(vals).length).toBeGreaterThan(0)
      for (const [value, meta] of Object.entries(vals)) {
        expect(meta.label.length).toBeGreaterThan(0)
        expect(meta.color).toMatch(/^#[0-9a-f]{6}$/)
        expect(`${domain}:${value}`).toBeTruthy()
      }
    }
  })
})

describe('StatusTag 渲染', () => {
  it('已知值渲染中文 label 与颜色', () => {
    const html = renderToStaticMarkup(<StatusTag domain="order" value="INSTALLING" />)
    expect(html).toContain('装维中')
    expect(html).toContain('#1677ff')
  })

  it('未注册域/未知值渲染 unknown(灰),不抛错', () => {
    expect(renderToStaticMarkup(<StatusTag domain="nope" value="X" />)).toContain('X')
    expect(renderToStaticMarkup(<StatusTag domain="order" value="???" />)).toContain('???')
  })

  it('快照:全枚举渲染无遗漏(出现 unknown 标记即失败)', () => {
    let count = 0
    for (const [domain, vals] of Object.entries(REGISTRY)) {
      for (const value of Object.keys(vals)) {
        const html = renderToStaticMarkup(<StatusTag domain={domain} value={value} />)
        expect(html).not.toContain('st-unknown')
        count++
      }
    }
    expect(count).toBeGreaterThanOrEqual(60)
  })
})
