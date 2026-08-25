// @vitest-environment jsdom
// vitest 默认 environment: 'node'(vite.config.ts),DOM 断言必须逐文件声明 jsdom。
import { describe, expect, it } from 'vitest'
import { setDocMeta } from './docMeta'

// 契约:设置 title+meta(name/property 双键),还原时新建元素移除、既有元素回滚旧值。
describe('setDocMeta', () => {
  it('设置并还原新建的 og meta', () => {
    document.title = 'before'
    const restore = setDocMeta('文章标题', { description: '摘要', 'og:title': '文章标题' })
    expect(document.title).toBe('文章标题')
    expect(document.head.querySelector('meta[name="description"]')?.getAttribute('content')).toBe('摘要')
    expect(document.head.querySelector('meta[property="og:title"]')?.getAttribute('content')).toBe('文章标题')
    restore()
    expect(document.title).toBe('before')
    expect(document.head.querySelector('meta[property="og:title"]')).toBeNull()
  })

  it('既有 meta 还原回滚而非移除', () => {
    const el = document.createElement('meta')
    el.name = 'description'
    el.content = '旧值'
    document.head.appendChild(el)
    const restore = setDocMeta('t', { description: '新值' })
    expect(el.content).toBe('新值')
    restore()
    expect(el.content).toBe('旧值')
    el.remove()
  })
})
