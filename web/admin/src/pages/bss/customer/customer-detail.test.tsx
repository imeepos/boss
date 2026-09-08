// @vitest-environment jsdom
// 客户详情抽屉回归(data-relations §6.1 销账):归属公司名直用行内 legalEntityName,
// 不再请求 /legal-entities 全量兜底;空值降级 —。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter } from 'react-router-dom'
import { CustomerDetailDrawer } from './index'
import type { CustomerRow } from './types'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { statusTags: {} },
    pages: {
      customer: {
        detail: '客户详情', chainTitle: '归属链', regionLabel: '区域', relTitle: '关联记录',
        relOrders: '订单', relBills: '账单', relPayments: '缴费', relLoadFail: '加载失败', relDrill: '下钻',
        columns: ['姓名', '手机号', '证件类型', '证件号', '实名状态', '服务状态', 'ID'],
      },
      company: { cancel: '关闭' },
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

const row = (over: Partial<CustomerRow>): CustomerRow => ({
  id: 7, name: 'Juan', phone: '09171234567', idType: '身份证', idNo: 'ID9001',
  realNameStatus: 'VERIFIED', serviceStatus: 'ACTIVE', addressId: 1, legalEntityId: 3,
  legalEntityName: 'Sphere North', regionId: 1, regionName: 'Metro Manila',
  createdAt: '2026-01-01T00:00:00Z', ...over,
})

let root: Root | null = null

afterEach(() => {
  mockFetch.mockReset()
  act(() => { root?.unmount() })
  document.body.innerHTML = ''
  root = null
})

async function mount(detail: CustomerRow) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<MemoryRouter><CustomerDetailDrawer detail={detail} onClose={() => {}} /></MemoryRouter>)
  })
}

describe('CustomerDetailDrawer 归属公司名', () => {
  it('直显行内 legalEntityName,且不再请求 /legal-entities', async () => {
    mockFetch.mockResolvedValue({ items: [] })
    await mount(row({}))
    const aside = document.body.querySelector('aside[role=dialog]')
    expect(aside?.textContent).toContain('Sphere North')
    expect(aside?.textContent).toContain('Metro Manila')
    const called = mockFetch.mock.calls.map((c) => c[0])
    expect(called).toContain('/orders')
    expect(called).toContain('/bills')
    expect(called).not.toContain('/legal-entities')
  })

  it('legalEntityName 为空降级 —,不回退 #id', async () => {
    mockFetch.mockResolvedValue({ items: [] })
    await mount(row({ legalEntityName: '' }))
    const aside = document.body.querySelector('aside[role=dialog]')
    expect(aside?.textContent).toContain('—→Metro Manila→Juan #7')
    expect(aside?.textContent).not.toContain('#3')
  })
})
