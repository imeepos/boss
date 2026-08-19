import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  activeServer,
  apiBaseUrl,
  listServers,
  newServerId,
  removeServer,
  setActiveServerId,
  upsertServer,
  type ServerConfig,
} from './serverConfig'

// 服务端配置存储:无内置默认;增删改查(含校验/重名);active 悬空回落 null。
// vitest 为 node 环境,用 Map 桩替代 localStorage。
function stubLocalStorage() {
  const map = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => void map.set(k, v),
    removeItem: (k: string) => void map.delete(k),
  })
}

describe('serverConfig', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('无配置:列表空、activeServer 为 null、apiBaseUrl 为相对占位(UI 门禁拦在前)', () => {
    stubLocalStorage()
    expect(listServers()).toEqual([])
    expect(activeServer()).toBeNull()
    expect(apiBaseUrl()).toBe('/api/admin/v1')
  })

  it('upsert 新增 + 启用 → apiBaseUrl 直连绝对地址(去尾斜杠)', () => {
    stubLocalStorage()
    const r = upsertServer({ name: '内网102', baseUrl: ' http://192.168.0.102:28080/ ' })
    if (!r.ok) throw new Error('should pass')
    setActiveServerId(r.servers[0].id)
    expect(apiBaseUrl()).toBe('http://192.168.0.102:28080/api/admin/v1')
  })

  it('upsert 编辑保持 id 与 active;校验失败不落库', () => {
    stubLocalStorage()
    const r = upsertServer({ name: 'a', baseUrl: 'http://a:1' })
    if (!r.ok) throw new Error('should pass')
    const id = r.servers[0].id
    setActiveServerId(id)
    const edit = upsertServer({ id, name: 'a2', baseUrl: 'http://a:2' })
    if (!edit.ok) throw new Error('should pass')
    expect(edit.servers[0]).toEqual({ id, name: 'a2', baseUrl: 'http://a:2' })
    expect(apiBaseUrl()).toBe('http://a:2/api/admin/v1')
    expect(upsertServer({ name: 'b', baseUrl: '' })).toEqual({ ok: false, error: 'url' })
    expect(upsertServer({ name: 'b', baseUrl: 'not-a-url' })).toEqual({ ok: false, error: 'url' })
    expect(upsertServer({ name: 'a2', baseUrl: 'http://x:1' })).toEqual({ ok: false, error: 'name' })
    expect(listServers()).toHaveLength(1)
  })

  it('删除生效中的配置 → active 清空回落 null', () => {
    stubLocalStorage()
    const r = upsertServer({ name: 'a', baseUrl: 'http://a:1' })
    if (!r.ok) throw new Error('should pass')
    const id = r.servers[0].id
    setActiveServerId(id)
    const del = removeServer(id)
    expect(del.activeCleared).toBe(true)
    expect(activeServer()).toBeNull()
    expect(apiBaseUrl()).toBe('/api/admin/v1')
  })

  it('active 指向不存在项 → activeServer 为 null', () => {
    stubLocalStorage()
    const one: ServerConfig = { id: 'srv-a', name: 'a', baseUrl: 'http://a:1' }
    listServers()
    localStorage.setItem('boss.servers', JSON.stringify([one]))
    localStorage.setItem('boss.server.active', 'srv-gone')
    expect(activeServer()).toBeNull()
  })

  it('无 localStorage 环境不抛错', () => {
    expect(listServers()).toEqual([])
    expect(apiBaseUrl()).toBe('/api/admin/v1')
  })

  it('newServerId 唯一', () => {
    expect(newServerId()).not.toBe(newServerId())
  })
})
