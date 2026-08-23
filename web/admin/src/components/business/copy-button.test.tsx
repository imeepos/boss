// CopyButton 回归:安全上下文优先 clipboard API;非安全上下文(http)降级 execCommand。
import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { CopyButton, copyText } from './feedback'
import { LocaleProvider } from '../../i18n/context'

describe('CopyButton', () => {
  it('渲染"复制"文案与传入内容无关的按钮', () => {
    const html = renderToStaticMarkup(
      <LocaleProvider><CopyButton text="ops_secret_123" /></LocaleProvider>,
    )
    expect(html).toContain('复制')
    expect(html).not.toContain('ops_secret_123') // 按钮不回显密钥本体
  })
})

describe('copyText', () => {
  const origNavigator = globalThis.navigator
  const origDocument = globalThis.document

  afterEach(() => {
    Object.defineProperty(globalThis, 'navigator', { value: origNavigator, configurable: true })
    Object.defineProperty(globalThis, 'document', { value: origDocument, configurable: true })
  })

  it('安全上下文:优先 navigator.clipboard.writeText', () => {
    const writeText = vi.fn(() => Promise.resolve())
    Object.defineProperty(globalThis, 'navigator', { value: { clipboard: { writeText } }, configurable: true })
    expect(copyText('abc')).toBe(true)
    expect(writeText).toHaveBeenCalledWith('abc')
  })

  it('http 降级:textarea + execCommand 复制后移除节点', () => {
    Object.defineProperty(globalThis, 'navigator', { value: {}, configurable: true })
    const removed: string[] = []
    type MockTa = { value: string; style: Record<string, string>; select: ReturnType<typeof vi.fn>; remove: () => void }
    const created: MockTa[] = []
    const doc = {
      createElement: () => {
        const ta: MockTa = { value: '', style: {}, select: vi.fn(), remove: () => { removed.push('ta') } }
        created.push(ta)
        return ta
      },
      body: { appendChild: vi.fn() },
      execCommand: vi.fn(() => true),
    }
    Object.defineProperty(globalThis, 'document', { value: doc, configurable: true })
    expect(copyText('abc')).toBe(true)
    expect(created).toHaveLength(1)
    expect(created[0].value).toBe('abc')
    expect(created[0].select).toHaveBeenCalled()
    expect(doc.body.appendChild).toHaveBeenCalled()
    expect(doc.execCommand).toHaveBeenCalledWith('copy')
    expect(removed).toEqual(['ta'])
  })
})