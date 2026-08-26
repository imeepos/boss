// @vitest-environment jsdom
// TaskList 焦点/可见性刷新回归:回到页面时自动重拉,保留手动刷新与筛选状态。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { ImportTaskList } from './TaskList'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    pages: {
      importer: {
        tasksTitle: '导入任务记录', taskKindFilter: '按类型筛选', taskOperatorFilter: '按操作人筛选',
        taskKindGeo: 'ISO 地理数据', taskKindAddr: '地址层级', entityNames: {},
        taskColumns: ['任务', '类型', '操作人', '总行数', '成功数', '失败数', '跳过数', '时间'],
        empty: '暂无数据', loadFail: '导入失败', refresh: '刷新',
      },
      audit: { refresh: '刷新' },
      company: { empty: '暂无数据' },
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

afterEach(() => {
  mockFetch.mockReset()
  document.body.innerHTML = ''
})

function mount(): Root {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  act(() => { root.render(<ImportTaskList />) })
  return root
}

describe('ImportTaskList visibility refresh', () => {
  it('初始挂载加载一次,获得焦点时再次加载', () => {
    mockFetch.mockResolvedValue({ items: [] })
    const root = mount()
    expect(mockFetch).toHaveBeenCalledTimes(1)
    act(() => { window.dispatchEvent(new Event('focus')) })
    expect(mockFetch).toHaveBeenCalledTimes(2)
    act(() => { root.unmount() })
  })

  it('页面恢复可见(visibilitychange)时加载', () => {
    mockFetch.mockResolvedValue({ items: [] })
    const root = mount()
    expect(mockFetch).toHaveBeenCalledTimes(1)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    act(() => { document.dispatchEvent(new Event('visibilitychange')) })
    expect(mockFetch).toHaveBeenCalledTimes(2)
    act(() => { root.unmount() })
  })
})