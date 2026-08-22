// 用户端配置 Tab 主键字段回归:每 tab 的 idKey 必须命中后端 SQL 别名,否则 ID 列会变 undefined。
// 对账源:internal/domain/customer/userdata/pg_lists.go 的 SELECT ... AS "..."。
import { describe, expect, it } from 'vitest'
import { TABS } from './tabs'

const idKeys = Object.fromEntries(TABS.map((t) => [t.key, t.idKey]))

describe('userdata tabs idKey', () => {
  it('notify 用 customerId(后端按客户聚合偏好)', () => {
    expect(idKeys.notify).toBe('customerId')
  })
  it('addons 用 addonId(后端 SELECT addon_id AS "addonId")', () => {
    expect(idKeys.addons).toBe('addonId')
  })
  it('coupons 用 couponId', () => {
    expect(idKeys.coupons).toBe('couponId')
  })
  it('topup 用 denomId', () => {
    expect(idKeys.topup).toBe('denomId')
  })
  it('faqs 用 faqId', () => {
    expect(idKeys.faqs).toBe('faqId')
  })
  it('guides 用 guideId', () => {
    expect(idKeys.guides).toBe('guideId')
  })
  it('invite-config 用 id', () => {
    expect(idKeys.invite).toBe('id')
  })
  it('路径端点前缀以 / 开头且无尾斜杠', () => {
    for (const t of TABS) {
      expect(t.path.startsWith('/')).toBe(true)
      expect(t.path.endsWith('/')).toBe(false)
    }
  })
  it('有 action 的 tab 必须同时给 actionPath', () => {
    for (const t of TABS) {
      if (t.action) {
        expect(typeof t.actionPath).toBe('string')
        expect(t.actionPath?.startsWith('/')).toBe(true)
      }
    }
  })
})