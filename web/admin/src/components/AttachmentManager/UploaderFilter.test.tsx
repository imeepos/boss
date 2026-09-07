// @vitest-environment jsdom
// UploaderFilter 行为回归:账号数据源懒加载(/accounts 一次)与失败重试;
// 未选类型禁用且不误发检索请求;非账号类型不拉账号列表。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { UploaderFilter, type UploaderFilterProps } from './UploaderFilter'

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

vi.mock('../../api/client', () => ({ apiFetch: vi.fn() }))

const mockFetch = (await import('../../api/client')).apiFetch as ReturnType<typeof vi.fn>

vi.mock('../../i18n', () => ({
  useT: () => ({
    attachmentManager: {
      allUploaders: '全部上传者',
      uploaderAccount: '账号',
      uploaderWorker: '师傅',
      uploaderCustomer: '客户',
      uploaderIdPlaceholder: '上传者 ID',
    },
    pages: { pickers: { common: { clear: '清除', retry: '重试' } } },
  }),
}))

afterEach(() => {
  mockFetch.mockReset()
  document.body.innerHTML = ''
})

function findByText(root: ParentNode, text: string): HTMLElement | null {
  return Array.from(root.querySelectorAll<HTMLElement>('button')).find((el) => el.textContent?.includes(text)) ?? null
}

function renderUi(props: Partial<UploaderFilterProps> = {}) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  const merged: UploaderFilterProps = { typeSel: 'account', onTypeChange: () => undefined, uid: '', onUidChange: () => undefined, ...props }
  act(() => { root.render(<UploaderFilter {...merged} />) })
  return { host, root, rerender: (next: Partial<UploaderFilterProps>) => act(() => { root.render(<UploaderFilter {...merged} {...next} />) }), flush: () => act(async () => {}) }
}

function idTrigger(host: ParentNode): HTMLButtonElement | null {
  return host.querySelector('[aria-label="上传者 ID"]')
}

describe('UploaderFilter', () => {
  it('account 类型懒加载 /accounts;worker 类型不拉账号列表', async () => {
    mockFetch.mockResolvedValue([])
    const ui = renderUi()
    await ui.flush()
    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(mockFetch).toHaveBeenCalledWith('/accounts')
    mockFetch.mockClear()
    ui.rerender({ typeSel: 'worker' })
    await ui.flush()
    expect(mockFetch.mock.calls.filter((c) => c[0] === '/accounts')).toHaveLength(0)
  })

  it('account 拉取失败就地可重试,重试重新拉取', async () => {
    mockFetch.mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce([])
    const ui = renderUi()
    await ui.flush()
    const retryBtn = findByText(ui.host, '重试')
    expect(retryBtn).not.toBeNull()
    await act(async () => { retryBtn!.click() })
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it('未选类型时选择器禁用且不发请求;account 模式可用', async () => {
    mockFetch.mockResolvedValue([])
    const ui = renderUi({ typeSel: '' })
    await ui.flush()
    const disabledTrigger = idTrigger(ui.host)
    expect(disabledTrigger).not.toBeNull()
    expect(disabledTrigger!.disabled).toBe(true)
    expect(mockFetch).not.toHaveBeenCalled()
    ui.rerender({ typeSel: 'account' })
    await ui.flush()
    expect(idTrigger(ui.host)!.disabled).toBe(false)
    expect(mockFetch).toHaveBeenCalledWith('/accounts')
  })
})
