// 认证配置纯逻辑:默认值草稿、secret 掩码提交、超时校验。
import { describe, expect, it } from 'vitest'
import { initDraft, payloadFor, timeoutError, type AuthFields } from './logic'

describe('initDraft', () => {
  it('空配置回默认值,secret 恒空', () => {
    const d = initDraft({} as AuthFields)
    expect(d['auth.cn.preloadTimeoutMs']).toBe('5000')
    expect(d['auth.cn.enabled']).toBe('false')
    expect(d['auth.my.provider']).toBe('none')
    expect(d['auth.cn.appSecret']).toBe('')
    expect(d['auth.my.apiKey']).toBe('')
  })

  it('接口值优先于默认值', () => {
    const d = initDraft({ 'auth.cn.appKey': { value: 'jk-1', hasValue: true } })
    expect(d['auth.cn.appKey']).toBe('jk-1')
  })
})

describe('payloadFor', () => {
  const loaded = initDraft({} as AuthFields)

  it('未变化的 key 不提交', () => {
    const p = payloadFor(['auth.cn.enabled', 'auth.cn.appKey'], { ...loaded }, loaded)
    expect(p).toEqual({})
  })

  it('变化 key 提交;secret 空串不提交、非空提交', () => {
    const draft = { ...loaded, 'auth.cn.appKey': 'new', 'auth.cn.appSecret': '' }
    const p = payloadFor(['auth.cn.appKey', 'auth.cn.appSecret'], draft, loaded)
    expect(p).toEqual({ 'auth.cn.appKey': 'new' })
    const p2 = payloadFor(['auth.cn.appSecret'], { ...loaded, 'auth.cn.appSecret': 's3cret' }, loaded)
    expect(p2).toEqual({ 'auth.cn.appSecret': 's3cret' })
  })
})

describe('timeoutError', () => {
  it('2000–10000 整数合法,其余非法', () => {
    expect(timeoutError('5000')).toBe(false)
    expect(timeoutError('2000')).toBe(false)
    expect(timeoutError('10000')).toBe(false)
    expect(timeoutError('1999')).toBe(true)
    expect(timeoutError('10001')).toBe(true)
    expect(timeoutError('abc')).toBe(true)
    expect(timeoutError('')).toBe(true)
    expect(timeoutError('5000.5')).toBe(true)
  })
})
