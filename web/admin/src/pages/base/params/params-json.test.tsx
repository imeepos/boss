// @vitest-environment jsdom
// 业务参数页 JSON 校验回归:形似 JSON 的值在 blur 时解析,失败给行:列 ErrorBanner;
// 非 JSON 值不校验,修正后横幅清除。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import ParamsPage from './index'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'

vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { copy: '复制', copied: '已复制', loading: '加载中', statusTags: {} },
    pages: {
      params: new Proxy({
        title: '业务参数', desc: '', searchPlaceholder: '搜索', allStatus: '全部状态',
        statusChanged: '已修改', statusOrigin: '原始', refresh: '刷新', cardTitle: '参数',
        hotUpdate: '热更新', colName: '参数', colValue: '值', colDesc: '说明', colOp: '操作',
        modified: '已修改', detail: '详情', empty: '暂无', save: '保存', saving: '保存中…',
        saved: '已保存 {count} 项', savePartial: '部分失败 {ok}/{fail}: {reason}', saveFail: '保存失败',
        resetForm: '重置', detailTitle: '详情', currentValue: '当前值', originValue: '原始值',
        close: '关闭', loadFail: '加载失败',
        jsonInvalid: '值不是合法 JSON',
        jsonInvalidAt: '值不是合法 JSON（第 {line} 行 第 {col} 列）',
      } as Record<string, unknown>, { get: (t, k) => (k in t ? t[k as string] : k) }),
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

let root: Root | null = null

afterEach(() => {
  mockFetch.mockReset()
  act(() => { root?.unmount() })
  document.body.innerHTML = ''
  root = null
})

async function mount() {
  mockFetch.mockImplementation((url: string) => {
    if (url === '/params') {
      return Promise.resolve({ items: [
        { key: 'ai.gateway', value: '{"ok":1}', desc: 'AI 网关' },
        { key: 'plain.key', value: 'hello', desc: '普通值' },
      ] })
    }
    return Promise.resolve({ items: [] })
  })
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => { root?.render(<ParamsPage />) })
}

function setInputByRow(rowKey: string, value: string): HTMLInputElement {
  const row = [...document.body.querySelectorAll('tbody tr')].find((tr) => tr.textContent?.includes(rowKey))
  const input = row?.querySelector('input') as HTMLInputElement
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
  setter?.call(input, value)
  input.dispatchEvent(new Event('input', { bubbles: true }))
  return input
}

const banners = () => [...document.body.querySelectorAll('div')]
  .filter((d) => typeof d.className === 'string' && d.className.includes('color-danger'))
  .map((d) => d.textContent)

describe('params JSON blur 校验', () => {
  it('非法 JSON 失焦后报 key + 行:列;非 JSON 值不校验', async () => {
    await mount()
    // 行展示名 = desc || key(paramLabel)
    const ai = setInputByRow('AI 网关', '{bad')
    await act(async () => { ai.dispatchEvent(new FocusEvent('focusout', { bubbles: true })) })
    expect(banners().join('')).toContain('ai.gateway: 值不是合法 JSON（第 1 行 第 2 列）')

    const plain = setInputByRow('普通值', 'not-json-at-all')
    await act(async () => { plain.dispatchEvent(new FocusEvent('focusout', { bubbles: true })) })
    expect(banners().join('')).not.toContain('plain.key')
  })

  it('修正为合法 JSON 后失焦,横幅清除', async () => {
    await mount()
    const ai = setInputByRow('AI 网关', '{bad')
    await act(async () => { ai.dispatchEvent(new FocusEvent('focusout', { bubbles: true })) })
    expect(banners().length).toBeGreaterThan(0)
    const fixed = setInputByRow('AI 网关', '{"ok":2}')
    await act(async () => { fixed.dispatchEvent(new FocusEvent('focusout', { bubbles: true })) })
    expect(banners().join('')).not.toContain('ai.gateway')
  })
})
