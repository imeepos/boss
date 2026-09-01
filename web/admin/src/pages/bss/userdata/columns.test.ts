// 用户端配置页列契约回归:每 tab 列组必须含 ID 主键列;操作列与 def.action 一一对应;
// 状态渲染走 StatusTag userdata 域。字段契约源:pg_lists.go 的 SELECT ... AS 别名。
import { describe, expect, it } from 'vitest'
import { TABS } from './tabs'
import { columnsFor, opLabel, type Labels } from './columns'

const lb: Labels = {
  cols: {
    customer: '客户', business: '业务通知', marketing: '营销通知', channel: '接收渠道',
    price: '价格', subscriberCount: '订阅数', amount: '面额', bonus: '赠送金额',
    expireAt: '有效期', category: '分类', question: '问题', title: '标题',
    inviteLink: '邀请链接', rewardAmount: '奖励金额',
  },
  channels: { app: 'App 推送' },
  nameCol: '名称', statusCol: '状态', opCol: '操作',
  disable: '停用', on: '启用', off: '停用', list: '上架', unlist: '下架',
}

describe('userdata per-tab columns', () => {
  it('每 tab 首列都是 ID 主键列', () => {
    for (const def of TABS) {
      const cols = columnsFor(def, lb, () => {})
      expect(cols[0].key, def.key).toBe('id')
    }
  })

  it('操作列与 def.action 一一对应(addons/coupons/faqs/guides 有,其余无)', () => {
    const withAction = TABS.filter((d) => d.action).map((d) => d.key).sort()
    expect(withAction).toEqual(['addons', 'coupons', 'faqs', 'guides'])
    for (const def of TABS) {
      const keys = columnsFor(def, lb, () => {}).map((c) => c.key)
      expect(keys.includes('op')).toBe(Boolean(def.action))
    }
  })

  it('每 tab 都有状态列或(notify)通知布尔列,字段名命中 SQL 别名', () => {
    const statusFields: Record<string, string[]> = {
      notify: ['business', 'marketing', 'channel'],
      addons: ['status'], coupons: ['status'],
      topup: ['status'], faqs: ['status'], guides: ['status'], invite: ['status'],
    }
    for (const def of TABS) {
      const cols = columnsFor(def, lb, () => {})
      const keys = cols.map((c) => c.key)
      for (const f of statusFields[def.key]) expect(keys, def.key).toContain(f)
    }
  })

  it('操作动词反映行当前态:on/active 现值给反向动作', () => {
    const addons = TABS.find((d) => d.key === 'addons')!
    expect(opLabel(addons, { status: 'on' }, lb)).toBe('下架')
    expect(opLabel(addons, { status: 'off' }, lb)).toBe('上架')
    const faqs = TABS.find((d) => d.key === 'faqs')!
    expect(opLabel(faqs, { active: true }, lb)).toBe('停用')
    expect(opLabel(faqs, { active: false }, lb)).toBe('启用')
    const coupons = TABS.find((d) => d.key === 'coupons')!
    expect(opLabel(coupons, {}, lb)).toBe('停用')
  })
})
