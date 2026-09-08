// @vitest-environment jsdom
// N4 编辑回显回归(2026-09-08 地址页实证):countryCode 显式传参时 country 初值即命中,
// boot 的 setCountry 同值 bailout;booted 若为 ref,回显效应永不重放,触发器不展开路径。
// 回归锁定:boot 完成必须是可观测 state,两种 boot 路径(显式国家/默认国家端点)都要回显。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { RegionCascadePicker } from './RegionCascadePicker'
import type { FetchLike } from './regionCascadeCore'

// React 18 act 环境标记:jsdom 下手动渲染需显式开启,否则告警刷屏。
;(globalThis as Record<string, unknown>).IS_REACT_ACT_ENVIRONMENT = true

vi.mock('../../i18n', () => ({
  useT: () => ({
    pages: {
      pickers: {
        common: { retry: '重试' },
        regionCascade: {
          country: '国家', levelSearch: '搜索本层', directSearch: '输入区划名或编码，跨层级直达',
          loading: '加载中…', empty: '暂无数据', loadFail: '区划加载失败', searchFail: '区划搜索失败',
          clear: '清除', aria: '选择国家与行政区划',
        },
      },
    },
  }),
}))

const LEVEL1 = [
  { code: 'PH-0100000000', displayName: 'PH-0100000000', level: 1, hasChildren: true, parentCode: '', countryCode: 'PH' },
  { code: 'PH-1300000000', displayName: 'PH-1300000000', level: 1, hasChildren: false, parentCode: '', countryCode: 'PH' },
]

function makeFetchImpl(defaultCountry: string): FetchLike {
  // FetchLike 的 T 由调用点实例化,桩按 unknown 兑现,类型面经此收窄为组件契约。
  const impl = vi.fn(async (path: string): Promise<unknown> => {
    if (path.startsWith('/geo/countries')) return [{ alpha2: 'PH', displayName: 'Philippines' }]
    if (path.startsWith('/geo/default-country')) return { countryCode: defaultCountry, configured: true }
    if (path.startsWith('/geo/subdivisions')) {
      const url = new URL('http://x' + path)
      const kw = url.searchParams.get('keyword') ?? ''
      const parent = url.searchParams.get('parentCode')
      if (kw) return LEVEL1.filter((r) => r.code === kw || r.displayName.includes(kw))
      if (parent === '') return LEVEL1
      return []
    }
    return null
  })
  return impl as unknown as FetchLike
}

async function renderPicker(props: Parameters<typeof RegionCascadePicker>[0]) {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => { root.render(<RegionCascadePicker {...props} />) })
  for (let i = 0; i < 4; i++) {
    await act(async () => { await Promise.resolve() })
  }
  return { host, root }
}

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

describe('RegionCascadePicker 编辑回显(N4)', () => {
  it('显式 countryCode 传参时,已存 value 反查链回显并出清除按钮', async () => {
    const { host } = await renderPicker({ value: 'PH-1300000000', countryCode: 'PH', fetchImpl: makeFetchImpl('PH') })
    const text = host.textContent ?? ''
    // 名称前缀取决于国家列表与反查的兑现次序,断言稳定不变量:链分隔 + 已存 code + 清除钮。
    expect(text).toContain(' / ')
    expect(text).toContain('PH-1300000000')
    expect(text).toContain('清除')
  })

  it('未传 countryCode 时经默认国家端点 boot,回显同样成立', async () => {
    const { host } = await renderPicker({ value: 'PH-1300000000', fetchImpl: makeFetchImpl('PH') })
    const text = host.textContent ?? ''
    expect(text).toContain(' / ')
    expect(text).toContain('PH-1300000000')
    expect(text).toContain('清除')
  })

  it('无 value 时不回显、不出清除按钮', async () => {
    const { host } = await renderPicker({ fetchImpl: makeFetchImpl('PH') })
    expect(host.textContent ?? '').not.toContain(' / ')
    expect(host.textContent ?? '').not.toContain('清除')
  })
})
