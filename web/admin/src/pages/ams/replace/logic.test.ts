// 换新单取消操作用例:仅 PENDING 显示取消;路径组装正确。
import { describe, expect, it } from 'vitest'
import { canCancelReplacement, cancelPath } from './logic'

describe('canCancelReplacement 状态显隐', () => {
  it('仅 PENDING 可取消,DOING/DONE/FAILED 一律不可', () => {
    for (const s of ['PENDING', 'DOING', 'DONE', 'FAILED']) {
      expect(canCancelReplacement(s)).toBe(s === 'PENDING')
    }
  })
})

describe('cancelPath', () => {
  it('组装 POST /replacements/{id}/cancel 路径', () => {
    expect(cancelPath(7)).toBe('/replacements/7/cancel')
  })
})
