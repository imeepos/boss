// 快捷搜索展示逻辑测试:各域命中主键与展示文案映射。
import { describe, expect, it } from 'vitest'
import { hitId, hitText } from './QuickSearch'
import type { CustomerHit, OrderHit, UserHit, WorkerHit } from '../../api/search'

describe('hitId', () => {
  it('按域取主键字段', () => {
    expect(hitId('customer', { id: 1 } as CustomerHit)).toBe(1)
    expect(hitId('user', { customerId: 2 } as UserHit)).toBe(2)
    expect(hitId('worker', { id: 3 } as WorkerHit)).toBe(3)
    expect(hitId('order', { id: 4 } as OrderHit)).toBe(4)
  })
})

describe('hitText', () => {
  it('客户:名称主行,客户码+手机号次行', () => {
    const t = hitText('customer', { name: '王先生', customerCode: 'C-1', phone: '13800001111' } as CustomerHit)
    expect(t.title).toBe('王先生')
    expect(t.sub).toBe('C-1 · 13800001111')
  })

  it('用户:名称主行,登录名+手机号次行', () => {
    const t = hitText('user', { name: '王先生', loginName: 'wang', phone: '13800001111' } as UserHit)
    expect(t.title).toBe('王先生')
    expect(t.sub).toBe('wang · 13800001111')
  })

  it('师傅:名称主行,工号+手机号次行', () => {
    const t = hitText('worker', { name: '李师傅', staffNo: 'W001', phone: '13900002222' } as WorkerHit)
    expect(t.title).toBe('李师傅')
    expect(t.sub).toBe('W001 · 13900002222')
  })

  it('订单:单号主行,客户+产品次行', () => {
    const t = hitText('order', { orderNo: 'ORD-1', customer: '王先生', product: '100M' } as OrderHit)
    expect(t.title).toBe('ORD-1')
    expect(t.sub).toBe('王先生 · 100M')
  })

  it('缺字段时次行仅保留有值部分', () => {
    const t = hitText('customer', { name: '王先生', customerCode: '', phone: '' } as CustomerHit)
    expect(t.sub).toBe('')
  })
})
