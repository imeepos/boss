import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { TabBar } from './tab-bar'

// 契约:平铺页签渲染激活态并使用设计令牌(设计系统 §2.1/§3)。
describe('TabBar', () => {
  const tabs = [{ key: 'a', label: '页签A' }, { key: 'b', label: '页签B' }]

  it('激活页签携带 aria-selected 与令牌高亮', () => {
    const html = renderToStaticMarkup(<TabBar tabs={tabs} value="a" onChange={() => {}} />)
    expect(html).toContain('aria-selected="true"')
    expect(html).toContain('var(--color-border-focus)')
    expect(html).toContain('页签A')
    expect(html).toContain('页签B')
  })

  it('非激活页签透明下边线', () => {
    const html = renderToStaticMarkup(<TabBar tabs={tabs} value="a" onChange={() => {}} />)
    expect(html).toContain('2px solid transparent')
    expect(html).toContain('var(--shell-side-border)')
  })

  it('extra 插槽渲染在页签行右侧(缺省不渲染容器)', () => {
    const withExtra = renderToStaticMarkup(
      <TabBar tabs={tabs} value="a" onChange={() => {}} extra={<button>刷新</button>} />,
    )
    expect(withExtra).toContain('刷新')
    expect(withExtra).toContain('shrink-0')
    const withoutExtra = renderToStaticMarkup(<TabBar tabs={tabs} value="a" onChange={() => {}} />)
    expect(withoutExtra).not.toContain('shrink-0')
  })
})
