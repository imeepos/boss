// AppDetail 纯函数回归:应用详情只显示本应用订阅的投递。
import { describe, expect, it } from 'vitest'
import { filterAppDeliveries } from './AppDetail'

describe('filterAppDeliveries', () => {
  const subs = [
    { id: 1, eventType: 'order.stage.done', endpointUrl: 'https://a.example/hook', status: 1, createdAt: '' },
    { id: 2, eventType: 'openplat.test', endpointUrl: 'https://a.example/hook', status: 1, createdAt: '' },
  ]
  const deliveries = [
    { id: 10, subscriptionId: 1, eventId: 'ORD-1:stage:5', eventType: 'order.stage.done', status: 1, attempts: 1, httpStatus: 200, lastError: '', createdAt: '' },
    { id: 11, subscriptionId: 9, eventId: 'other-app', eventType: 'order.stage.done', status: 0, attempts: 0, httpStatus: 0, lastError: '', createdAt: '' },
    { id: 12, subscriptionId: 2, eventId: 'test-1', eventType: 'openplat.test', status: 2, attempts: 6, httpStatus: 500, lastError: 'http 500', createdAt: '' },
  ]

  it('只保留属于本应用订阅的投递', () => {
    expect(filterAppDeliveries(deliveries, subs).map((d) => d.id)).toEqual([10, 12])
  })

  it('无订阅时为空', () => {
    expect(filterAppDeliveries(deliveries, [])).toEqual([])
  })
})
