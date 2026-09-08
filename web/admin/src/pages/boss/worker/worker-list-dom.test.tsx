// @vitest-environment jsdom
// 师傅列表 DOM 回归(data-relations §6.2 销账):班组列直显行内 groupName、
// 负责区域优先主区域名 regionName;取不到降级 —,不再渲染 #id 裸编号。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter } from 'react-router-dom'
import WorkerPage from './index'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { loading: '加载中', statusTags: {} },
    pages: {
      workerPage: new Proxy({
        title: '师傅管理', desc: '', loadFail: '加载失败', searchPlaceholder: '搜索',
        columns: ['工号', '姓名', '班组', '负责区域', '手机号', '状态', '入职时间', '操作'],
        empty: '暂无数据', allMembers: '全部成员', teamTitle: '装维队', memberCount: '成员数',
        captain: '队长', captainEmpty: '未指定', teamOps: '队伍操作', editTeam: '编辑',
        perfBtn: '业绩', disband: '解散', newWorker: '新增师傅', newTeam: '添加装维队',
        needSelectGroup: '请先选择装维队', pickWorker: '选择师傅', pickWorkerPlaceholder: '检索师傅',
        addedToGroup: '已加入', addToGroupHint: '调队', actionFail: '操作失败',
        active: '在职', left: '离职', detail: '详情', editRegions: '负责区域',
        setCaptain: '设为队长', transfer: '调队',
        rangeText: '', prev: '上一页', next: '下一页', perPage: '条/页', jumpText: '跳至', pageUnit: '页',
      } as Record<string, unknown>, { get: (t, k) => (k in t ? t[k as string] : k) }),
      audit: { refresh: '刷新' },
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

const groups = { items: [{ id: 5, legalEntityId: 1, code: 'T1', name: '一队', leaderId: 0, leaderName: '', memberCount: 1 }] }
const worker = (over: Record<string, unknown>) => ({
  id: 9, staffNo: 'W001', name: '李四', groupId: 5, groupName: '一队',
  regionId: 2, regionName: 'Cebu', regionIds: [2, 9], phone: '09170000000',
  status: 1, joinedAt: '2026-01-01T00:00:00Z', ...over,
})
const regions = [{ id: 2, name: 'Cebu' }, { id: 9, name: 'Davao' }]

function route(url: string) {
  if (url === '/worker-groups') return Promise.resolve(groups)
  if (url === '/workers') return Promise.resolve({ items: workers })
  if (url === '/regions') return Promise.resolve(regions)
  return Promise.resolve({ items: [] })
}

let workers: Record<string, unknown>[] = []
let root: Root | null = null

afterEach(() => {
  mockFetch.mockReset()
  act(() => { root?.unmount() })
  document.body.innerHTML = ''
  root = null
  workers = []
})

async function mount() {
  mockFetch.mockImplementation((url: string) => route(url))
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<MemoryRouter><WorkerPage /></MemoryRouter>)
  })
}

const cells = () => [...document.body.querySelectorAll('tbody tr')].map((tr) => tr.textContent ?? '')

describe('师傅列表 groupName/regionName 直显', () => {
  it('班组列与负责区域列渲染人读名,无 #id 裸编号', async () => {
    workers = [worker({})]
    await mount()
    const row = cells()[0]
    expect(row).toContain('一队')
    expect(row).toContain('Cebu、Davao')
    expect(row).not.toContain('#5')
    expect(row).not.toContain('#9')
  })

  it('行内 groupName 为空降级 —;regionName 缺失时经 /regions 映射兜底', async () => {
    workers = [worker({ groupName: '', regionName: '' })]
    await mount()
    const row = cells()[0]
    expect(row).toContain('Cebu、Davao')
    expect(row).toMatch(/W001李四—Cebu、Davao/)
  })
})
