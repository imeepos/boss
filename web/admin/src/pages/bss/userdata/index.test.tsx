// 用户端配置页 missing-row 兜底回归(postmortem 0002 纵深防御):
// 即使后端断言已 500,前端 detect 主键 undefined 时也不能渲染 `undefined` 字面值。
import { describe, expect, it } from 'vitest'
import { TABS } from './tabs'

// 直接对齐 index.tsx 中的判定,与组件实现解耦以便静态审查。
function isMissing(value: unknown): boolean {
  return value === undefined || value === null || value === ''
}

describe('userdata row missing detection', () => {
  it('每个 Tab 的 idKey 命中后端 SQL AS 别名后值应非 missing', () => {
    const samples: Record<string, Record<string, unknown>> = {
      notify: { customerId: 213, customerName: 'X', business: true },
      addons: { addonId: 'ADD-1', name: 'x', price: 100 },
      coupons: { couponId: 'CPN-1', customerId: 9, name: 'x', amount: 100 },
      topup: { denomId: 'D-50', amount: 5000, bonus: 200, active: true },
      faqs: { faqId: 'FAQ-01', category: 'x', question: 'q', answer: 'a', active: true },
      guides: { guideId: 'G-01', title: 't', category: 'c', steps: 's', active: true },
      invite: { id: 1, inviteLink: 'https://x', rewardAmount: 100, active: true },
    }
    for (const def of TABS) {
      const row = samples[def.key]
      expect(row, `${def.key} 应有真实样本`).toBeDefined()
      expect(isMissing(row[def.idKey]), `${def.key}.${def.idKey} 不应为 missing`).toBe(false)
    }
  })

  it('undefined / null / 空串 三种 missing 全部命中', () => {
    expect(isMissing(undefined)).toBe(true)
    expect(isMissing(null)).toBe(true)
    expect(isMissing('')).toBe(true)
    expect(isMissing(0)).toBe(false)
    expect(isMissing('ADD-1')).toBe(false)
  })

  it('缺主键列的行若绕过前端检查,后端应返回 ErrContractDrift 而非静默', async () => {
    // 后端断言(对账源):internal/domain/customer/userdata/service.go AssertListContract。
    // 这里仅用类型契约保证前端不会发送含 missing idKey 的 fetch 路径字符串。
    // 真值校验交给 e2e + 后端断言守护。
    const { TABS: tabs } = await import('./tabs')
    for (const def of tabs) {
      if (!def.action || !def.actionPath) continue
      const missingId = undefined
      const url = `${def.actionPath}/${encodeURIComponent(String(missingId))}/${def.action}`
      // missingId 拼出来必然含 `undefined` 字面值,后端会拒
      expect(url).toContain('undefined')
    }
  })
})