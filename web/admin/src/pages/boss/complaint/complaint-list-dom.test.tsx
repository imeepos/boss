// @vitest-environment jsdom
// 报障列表 DOM 回归(data-relations §6.2 同批销账):客户列直显 customerName
// (customerId 编号并列保留);空名降级 —。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { ConfirmProvider } from '../../../components/ConfirmDialog'
import ComplaintPage from './index'
import type { ComplaintRow } from '../types'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: {
      loading: '加载中',
      statusTags: { 'complaint.OPEN': '待受理' },
      confirmDialog: { title: '确认', ok: '确定', cancel: '取消' },
    },
    pages: {
      complaintPage: new Proxy({
        title: '报障与投诉', desc: '', loadFail: '加载失败', actionFail: '操作失败',
        columns: ['工单号', '客户', '订单', '类型', '状态', '操作'],
        types: { NETWORK_FAULT: '网络故障' }, empty: '暂无数据',
        accept: '受理', close: '关单', closeConfirm: '确认关单', closeOk: '已关单 {no}', acceptOk: '已受理 {no}',
        eventsBtn: '轨迹', eventsTitle: '事件轨迹', eventsColumns: ['时间', '事件', '状态', '备注'], eventsEmpty: '暂无事件',
        rangeText: '', prev: '上一页', next: '下一页', perPage: '条/页', jumpText: '跳至', pageUnit: '页',
      } as Record<string, unknown>, { get: (t, k) => (k in t ? t[k as string] : k) }),
      audit: { refresh: '刷新' },
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

const row = (over: Partial<ComplaintRow>): ComplaintRow => ({
  id: 1, ticketNo: 'TK-001', customerId: 7, customerName: 'Juan', orderId: 3,
  legalEntityId: 2, legalEntityName: 'Sphere North', type: 'NETWORK_FAULT', status: 'OPEN',
  ...over,
})

const metrics = { openCount: 1, processingCount: 0, closedCount: 0, slaBreachedOpen: 0, slaOnTimeClosed: 0, slaOverdueClosed: 0, avgCloseHours: 1.5 }

let rows: ComplaintRow[] = []
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
    if (url === '/complaints') return Promise.resolve(rows)
    if (url === '/complaint-metrics') return Promise.resolve(metrics)
    return Promise.resolve({ items: [] })
  })
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<ConfirmProvider><ComplaintPage /></ConfirmProvider>)
  })
}

describe('报障列表 customerName 直显', () => {
  it('客户列渲染 customerName 且 customerId 编号并列', async () => {
    rows = [row({})]
    await mount()
    const first = document.body.querySelector('tbody tr')?.textContent ?? ''
    expect(first).toContain('Juan')
    expect(first).toContain('#7')
  })

  it('customerName 为空降级 —,customerId=0 不渲染空编号', async () => {
    rows = [row({ customerName: '', customerId: 0 })]
    await mount()
    const first = document.body.querySelector('tbody tr')?.textContent ?? ''
    expect(first).toContain('—')
    expect(first).not.toContain('#0')
  })
})
