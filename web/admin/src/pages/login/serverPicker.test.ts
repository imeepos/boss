import { afterEach, describe, expect, it, vi } from 'vitest'
import { initialPickerState, pickServer } from './serverPicker'
import { activeServer, apiBaseUrl, upsertServer } from '../../lib/serverConfig'

// 登录页服务端选择:选择持久化(刷新不丢),无默认环境(空态由 App 门禁兜住)。
function stubLocalStorage() {
  const map = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => void map.set(k, v),
    removeItem: (k: string) => void map.delete(k),
  })
}

describe('serverPicker', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('无配置:列表空且 activeId 为空串(App 门禁会先弹框)', () => {
    stubLocalStorage()
    expect(initialPickerState()).toEqual({ servers: [], activeId: '' })
  })

  it('选择持久化:pick 后模拟刷新仍命中,基址随切', () => {
    stubLocalStorage()
    const r = upsertServer({ name: '内网102', baseUrl: 'http://192.168.0.102:28080' })
    if (!r.ok) throw new Error('should pass')
    const r2 = upsertServer({ name: '本机', baseUrl: 'http://127.0.0.1:28080' })
    if (!r2.ok) throw new Error('should pass')
    const local = r2.servers.find((it) => it.name === '本机')!
    expect(pickServer(local.id)).toBe(local.id)
    expect(initialPickerState().activeId).toBe(local.id)
    expect(apiBaseUrl()).toBe('http://127.0.0.1:28080/api/v1')
    expect(activeServer()?.name).toBe('本机')
  })
})
