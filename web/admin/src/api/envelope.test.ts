import { describe, expect, it } from 'vitest'
import { ApiError, unwrap } from './envelope'

// 契约:HTTP 恒 200,业务成败看 envelope.code;0=OK 取 data,401=未授权,其余抛 ApiError。
describe('unwrap envelope', () => {
  it('code=0 返回 data 负载', () => {
    expect(unwrap({ code: 0, msg: 'ok', data: { x: 1 } })).toEqual({ x: 1 })
  })

  it('code=401 抛 UNAUTHORIZED 供守卫登出', () => {
    try {
      unwrap({ code: 401, msg: 'token 过期', data: null })
      throw new Error('should throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError)
      expect((e as ApiError).code).toBe(401)
      expect((e as ApiError).unauthorized).toBe(true)
    }
  })

  it('其他业务码抛携带 code+msg 的 ApiError', () => {
    try {
      unwrap({ code: 1003, msg: '参数错误', data: null })
      throw new Error('should throw')
    } catch (e) {
      expect((e as ApiError).code).toBe(1003)
      expect((e as ApiError).message).toBe('参数错误')
      expect((e as ApiError).unauthorized).toBe(false)
    }
  })

  it('data 缺失时返回 null', () => {
    expect(unwrap({ code: 0, msg: 'ok' })).toBeNull()
  })
})
