// @vitest-environment jsdom
// 存储配置「测试连接」回归(data-relations §6.3 销账):SubmitButton 状态机 +
// ErrorBanner 展示 POST /storage-config/test 结果,对齐 auth/sms 等配置页交互。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import StorageConfigPage from './index'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
import { apiFetch } from '../../../api/client'
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { copy: '复制', copied: '已复制' },
    pages: {
      storageconfig: new Proxy({
        title: 'MinIO 存储配置', desc: '', cardTitle: 'MinIO 连接', cardDesc: '',
        configured: '已配置', notConfigured: '未配置', endpoint: '服务地址', endpointHint: '',
        bucket: '存储桶', accessKey: 'Access Key', secretKey: 'Secret Key', useSSL: '启用 HTTPS',
        enabled: '已启用', disabled: '未启用', save: '保存配置', saving: '保存中…', saved: '已保存',
        saveFail: '保存失败', loadFail: '加载失败', retry: '重试',
        rotateSecret: '轮换', rotateSecretDesc: '', newSecret: '新密码', newSecretPlaceholder: '',
        rotate: '确认轮换', rotating: '轮换中…', rotated: '已轮换', rotateFail: '轮换失败', cancel: '取消',
        testBtn: '测试连接', testing: '测试中…', testOk: '连接成功', testFail: '连接失败',
        testHint: '对已保存配置发起真实 MinIO 探测',
      } as Record<string, unknown>, { get: (t, k) => (k in t ? t[k as string] : k) }),
    },
  }),
}))

const mockFetch = apiFetch as ReturnType<typeof vi.fn>

let testResult: { ok: boolean; latencyMs: number; message?: string } | null = null
let root: Root | null = null

afterEach(() => {
  mockFetch.mockReset()
  act(() => { root?.unmount() })
  document.body.innerHTML = ''
  root = null
  testResult = null
})

async function mount() {
  mockFetch.mockImplementation((url: string, init?: { method?: string }) => {
    if (url === '/storage-config' && !init?.method) return Promise.resolve({ fields: { 'minio.endpoint': { value: '192.168.0.102:29000', hasValue: true } } })
    if (url === '/storage-config/test') return Promise.resolve(testResult)
    return Promise.resolve({})
  })
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<StorageConfigPage />)
  })
}

const buttonByLabel = (label: string) =>
  [...document.body.querySelectorAll('button')].find((b) => b.textContent === label)

describe('storageconfig 测试连接', () => {
  it('探测失败:message 经 ErrorBanner 展示,按钮转失败态', async () => {
    testResult = { ok: false, latencyMs: 4, message: 'dial tcp 192.168.0.102:29000: connect refused' }
    await mount()
    const btn = buttonByLabel('测试连接')
    expect(btn).toBeDefined()
    await act(async () => { btn?.click() })
    // ErrorBanner:danger 色横幅块内展示可复制失败原因
    const banner = [...document.body.querySelectorAll('div')].find((d) => d.className.includes('color-danger'))
    expect(document.body.textContent).toContain('dial tcp 192.168.0.102:29000: connect refused')
    expect(buttonByLabel('连接失败')).toBeDefined()
    expect(banner?.textContent).toContain('connect refused')
  })

  it('探测成功:按钮转成功态并携带时延,无错误横幅', async () => {
    testResult = { ok: true, latencyMs: 12 }
    await mount()
    const btn = buttonByLabel('测试连接')
    await act(async () => { btn?.click() })
    expect(mockFetch).toHaveBeenCalledWith('/storage-config/test', { method: 'POST' })
    expect(buttonByLabel('连接成功')).toBeDefined()
    expect(document.body.textContent).not.toContain('connect refused')
  })
})
