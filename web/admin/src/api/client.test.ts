// 复测 client.ts:body 传字符串时不要再 JSON.stringify 一遍。
import { describe, it, expect } from 'vitest'

describe('apiFetch body 序列化', () => {
  it('body 是字符串时不再二次 stringify', async () => {
    const calls: Array<{ url: string; init: RequestInit }> = []
    const realFetch = globalThis.fetch
    globalThis.fetch = (async (url: string | URL | Request, init?: RequestInit) => {
      calls.push({ url: String(url), init: init ?? {} })
      return new Response('{"code":0,"data":{"ok":true},"msg":"ok"}', {
        status: 200, headers: { 'Content-Type': 'application/json' },
      })
    }) as typeof fetch
    const { apiFetch } = await import('./client')
    await apiFetch('/x', { method: 'POST', body: '{"name":"abc"}' })
    globalThis.fetch = realFetch
    expect(calls).toHaveLength(1)
    expect(calls[0].init.body).toBe('{"name":"abc"}')
  })

  it('body 是对象时正常 stringify', async () => {
    const calls: Array<{ url: string; init: RequestInit }> = []
    const realFetch = globalThis.fetch
    globalThis.fetch = (async (url: string | URL | Request, init?: RequestInit) => {
      calls.push({ url: String(url), init: init ?? {} })
      return new Response('{"code":0,"data":{},"msg":"ok"}', {
        status: 200, headers: { 'Content-Type': 'application/json' },
      })
    }) as typeof fetch
    const { apiFetch } = await import('./client')
    await apiFetch('/x', { method: 'POST', body: { name: 'abc' } })
    globalThis.fetch = realFetch
    expect(calls).toHaveLength(1)
    expect(calls[0].init.body).toBe('{"name":"abc"}')
  })
})
