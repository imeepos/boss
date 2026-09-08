// ToolbarButton className 追加回归(W3 aqp 批次):调用方布局类与默认类 cn 合并共存,
// 支撑 OrderItemsEditor 等网格场景去掉 wrapper div 直挂 col-span/w-full。
import { describe, it, expect } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ToolbarButton } from './page-head'

describe('ToolbarButton', () => {
  it('className 追加类与默认类合并共存', () => {
    const html = renderToStaticMarkup(
      <ToolbarButton onClick={() => {}} className="col-span-1 w-full">删行</ToolbarButton>,
    )
    expect(html).toContain('col-span-1')
    expect(html).toContain('w-full')
    expect(html).toContain('h-8')
  })

  it('primary 态同样支持 className,品牌实底不变', () => {
    const html = renderToStaticMarkup(
      <ToolbarButton primary onClick={() => {}} className="w-full">新增</ToolbarButton>,
    )
    expect(html).toContain('w-full')
    expect(html).toContain('bg-[var(--shell-fab-bg)]')
  })
})
