// @vitest-environment jsdom
// 师傅详情抽屉 DOM 回归(data-relations §6.2 销账):主档班组/区域直显详情行内
// groupName/regionName,不再经 /worker-groups、/regions 全量映射,空值降级 —。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { WorkerDetailDrawer } from './worker-detail-drawer'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { loading: '加载中', statusTags: {} },
    pages: {
      workerPage: new Proxy({
        detailTitle: '师傅详情', loadFail: '加载失败', cancel: '关闭', empty: '暂无数据',
        active: '在职', left: '离职',
        d: {
          dPhone: '手机', dRegion: '区域', dGroup: '班组', dJoinedAt: '入职', dLeftAt: '离职',
          dStatTickets: '在途工单', dStatTotalTickets: '工单总数', dStatMessages: '消息',
          dStatFeedbacks: '评价', dYes: '是', dNo: '否', dSeeAll: '查看全部', dCollapse: '收起',
        },
        sectionNames: { tickets: '工单', messages: '消息', performances: '绩效', feedbacks: '评价' },
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

async function mount(id: number) {
  mockFetch.mockImplementation((url: string) => {
    if (url === `/workers/${id}`) {
      return Promise.resolve({
        id, staffNo: 'W001', name: '李四', groupId: 5, groupName: '一队',
        regionId: 2, regionName: regionNameValue, phone: '09170000000', status: 1,
        joinedAt: '2026-01-01T00:00:00Z',
      })
    }
    return Promise.resolve({ items: [] })
  })
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<WorkerDetailDrawer id={id} onClose={() => {}} />)
  })
}

let regionNameValue = 'Cebu'

describe('WorkerDetailDrawer 班组/区域名直显', () => {
  it('主档渲染 groupName/regionName,且不再请求 /worker-groups 与 /regions', async () => {
    regionNameValue = 'Cebu'
    await mount(9)
    const aside = document.body.querySelector('aside[role=dialog]')
    expect(aside?.textContent).toContain('一队')
    expect(aside?.textContent).toContain('Cebu')
    expect(aside?.textContent).not.toContain('#5')
    expect(aside?.textContent).not.toContain('#2')
    const called = mockFetch.mock.calls.map((c) => c[0])
    expect(called).toContain('/workers/9')
    expect(called).not.toContain('/worker-groups')
    expect(called).not.toContain('/regions')
  })

  it('详情行内名字为空降级 —', async () => {
    regionNameValue = ''
    await mount(9)
    const aside = document.body.querySelector('aside[role=dialog]')
    expect(aside?.textContent).toContain('区域 —')
    expect(aside?.textContent).toContain('班组 一队')
  })
})
