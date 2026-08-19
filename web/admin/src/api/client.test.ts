import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiFetch, setAuthToken } from './client'

// 契约:单通道 fetch 封装——带 token、解 envelope、非 2xx 抛 ApiError。
describe('apiFetch', () => {
  afterEach(() => setAuthToken(null))

  it('POST JSON 并解 envelope', async () => {
    const mock = vi.fn(async () =>
      new Response(JSON.stringify({ code: 0, msg: 'ok', data: { token: 't1' } }), { status: 200 }),
    )
    vi.stubGlobal('fetch', mock)
    const data = await apiFetch<{ token: string }>('/auth/login', {
      method: 'POST',
      body: { username: 'a', password: 'b' },
    })
    expect(data).toEqual({ token: 't1' })
    const [url, init] = mock.mock.calls[0] as unknown as [string, RequestInit]
    // 无 localStorage(测试环境)→ 无已配置服务端,相对前缀仅为兜死占位;实际 UI 有服务端门禁。
    expect(url).toBe('/api/admin/v1/auth/login')
    expect(init.method).toBe('POST')
    expect(init.body).toBe(JSON.stringify({ username: 'a', password: 'b' }))
    vi.unstubAllGlobals()
  })

  it('携带 Authorization 头', async () => {
    setAuthToken('jwt-1')
    const mock = vi.fn(async () =>
      new Response(JSON.stringify({ code: 0, msg: 'ok', data: {} }), { status: 200 }),
    )
    vi.stubGlobal('fetch', mock)
    await apiFetch('/auth/me')
    const [, init] = mock.mock.calls[0] as unknown as [string, RequestInit]
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer jwt-1')
    vi.unstubAllGlobals()
  })

  it('业务码非 0 抛 ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn(async () =>
      new Response(JSON.stringify({ code: 1001, msg: '账号或密码错误', data: null }), { status: 200 }),
    ))
    await expect(apiFetch('/auth/login', { method: 'POST', body: {} })).rejects.toThrow('账号或密码错误')
    vi.unstubAllGlobals()
  })

  it('HTTP 非 2xx 抛 ApiError(code=httpStatus)', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('bad gateway', { status: 502 })))
    await expect(apiFetch('/auth/me')).rejects.toMatchObject({ code: 502 })
    vi.unstubAllGlobals()
  })
})
