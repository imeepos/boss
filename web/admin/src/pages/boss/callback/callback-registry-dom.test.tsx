// @vitest-environment jsdom
// 激活回调页回归(ux-final-audit §四 registry 收编):result 裸枚举改 StatusTag
// domain="callbackResult" 渲染,标签走三语字典。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { ConfirmProvider } from '../../../components/ConfirmDialog'
import CallbackPage from './index'
import type { ActivationCallbackRow } from '../types'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: {
      loading: '加载中',
      statusTags: { 'callbackResult.SUCCESS': '成功', 'callbackResult.FAILED': '失败' },
      confirmDialog: { title: '确认', ok: '确定', cancel: '取消' },
    },
    pages: {
      callbackPage: new Proxy({
        title: '激活回调', desc: '', loadFail: '加载失败', actionFail: '操作失败',
        columns: ['ID', '订单', '结果', '重试次数', '操作'],
        retry: '重试', retryConfirm: '确认重试', toastRetryOk: '已重试', empty: '暂无数据',
        rangeText: '', prev: '上一页', next: '下一页', perPage: '条/页', jumpText: '跳至', pageUnit: '页',
      } as Record<string, unknown>, { get: (t, k) => (k in t ? t[k as string] : k) }),
      audit: { refresh: '刷新' },
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

let rows: ActivationCallbackRow[] = []
let root: Root | null = null

afterEach(() => {
  mockFetch.mockReset()
  act(() => { root?.unmount() })
  document.body.innerHTML = ''
  root = null
  rows = []
})

async function mount() {
  mockFetch.mockImplementation((url: string) => {
    if (url === '/activation-callbacks') return Promise.resolve(rows)
    return Promise.resolve({ items: [] })
  })
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<ConfirmProvider><CallbackPage /></ConfirmProvider>)
  })
}

describe('激活回调结果 StatusTag 渲染', () => {
  it('SUCCESS/FAILED 渲染注册标签,不再裸贴英文枚举', async () => {
    rows = [
      { id: 1, orderId: 11, result: 'SUCCESS', retries: 0 },
      { id: 2, orderId: 12, result: 'FAILED', retries: 2 },
    ]
    await mount()
    const tags = [...document.body.querySelectorAll('tbody .st-tag')].map((el) => el.textContent)
    expect(tags).toEqual(['成功', '失败'])
    expect(document.body.textContent).not.toContain('SUCCESS')
    expect(document.body.textContent).not.toContain('FAILED')
  })
})
