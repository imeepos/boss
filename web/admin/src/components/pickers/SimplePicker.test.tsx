// @vitest-environment jsdom
// W3 基座收尾回归(2026-09-08,selector-audit §一.1 ①):
// 1) Dropdown 新增 onOpenChange 仅在开合过渡触发(mount 不触发),点击/键盘开、Esc/外点关全覆盖;
// 2) SimplePicker 服务端源重开浮层时复位检索——修「输入框已清而列表残留旧检索结果」窗口。
import { beforeAll, afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import type { ReactElement } from 'react'
import { Dropdown } from '../Dropdown'
import { SimplePicker } from './SimplePicker'

;(globalThis as Record<string, unknown>).IS_REACT_ACT_ENVIRONMENT = true

vi.mock('../../i18n', () => ({
  useT: () => ({
    pages: {
      pickers: {
        common: { loading: '加载中…', empty: '暂无数据', retry: '重试', clear: '清空' },
      },
    },
  }),
}))

async function renderEl(el: ReactElement) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => { root.render(el) })
  return { host, root }
}

const flush = async () => {
  for (let i = 0; i < 4; i++) await act(async () => { await Promise.resolve() })
}

const typeInput = (input: HTMLInputElement, v: string) => {
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
  setter?.call(input, v)
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

const pressKey = (el: Element, key: string) =>
  el.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }))

beforeAll(() => {
  // jsdom 未实现 scrollIntoView(活动项滚动效应),桩掉即可。
  Element.prototype.scrollIntoView = vi.fn()
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

describe('Dropdown onOpenChange(W3 基座收尾)', () => {
  it('仅在开合过渡触发:mount 静默,点击/键盘开 true,Esc/外点关 false', async () => {
    const spy = vi.fn()
    const { host } = await renderEl(
      <Dropdown value='' options={[{ value: 'a', label: 'A' }]} onChange={() => {}}
        ariaLabel='demo' searchable remote onKeywordChange={() => {}} onOpenChange={spy} />,
    )
    expect(spy).not.toHaveBeenCalled()
    const trigger = host.querySelector('button[aria-haspopup=listbox]') as HTMLButtonElement
    await act(async () => { trigger.click() })
    expect(spy).toHaveBeenLastCalledWith(true)
    const input = host.querySelector('input') as HTMLInputElement
    await act(async () => { pressKey(input, 'Escape') })
    expect(spy).toHaveBeenLastCalledWith(false)
    // 键盘开合(触发器 ArrowDown)同样通知
    await act(async () => { pressKey(trigger, 'ArrowDown') })
    expect(spy).toHaveBeenLastCalledWith(true)
    // 点击外部关闭(document mousedown)
    await act(async () => { document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true })) })
    expect(spy).toHaveBeenLastCalledWith(false)
    expect(spy).toHaveBeenCalledTimes(4)
  })
})

describe('SimplePicker 服务端源重开复位(W3 修残留窗口)', () => {
  it('输入关键字后关开浮层:重开即重发首屏检索且输入框已清', async () => {
    const fetcher = vi.fn(async (kw: string) => [{ value: 'opt-' + (kw || 'all'), label: '结果:' + (kw || '全部') }])
    const { host } = await renderEl(
      <SimplePicker value='' onChange={() => {}} search={fetcher} debounceMs={0}
        ariaLabel='物料' searchPlaceholder='检索' />,
    )
    await flush()
    expect(fetcher.mock.calls.map((c) => c[0])).toEqual([''])
    const trigger = host.querySelector('button[aria-haspopup=listbox]') as HTMLButtonElement
    await act(async () => { trigger.click() })
    await flush()
    const input = host.querySelector('input') as HTMLInputElement
    await act(async () => { typeInput(input, 'ab') })
    await act(async () => { await new Promise((r) => setTimeout(r, 10)) })
    await flush()
    expect(fetcher.mock.calls.map((c) => c[0])).toEqual(['', 'ab'])
    // Esc 关闭 → 重开:hook 旧关键字被复位,立即重发首屏检索
    await act(async () => { pressKey(input, 'Escape') })
    await act(async () => { trigger.click() })
    await act(async () => { await new Promise((r) => setTimeout(r, 10)) })
    await flush()
    expect(fetcher.mock.calls.map((c) => c[0])).toEqual(['', 'ab', ''])
    const reopenedInput = host.querySelector('input') as HTMLInputElement
    expect(reopenedInput.value).toBe('')
  })
})
