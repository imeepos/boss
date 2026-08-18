// 业务参数/审计日志筛选逻辑用例。
import { describe, expect, it } from 'vitest'
import { filterParams, type BizParam } from './logic'
import { filterAuditLogs, type AuditLog } from '../audit/logic'

const params: BizParam[] = [
  { key: 'arrears.threshold', value: '100', desc: '欠费判定阈值' },
  { key: 'reserve.ttl', value: '48', desc: '端口预占小时数' },
]

describe('filterParams', () => {
  it('关键字命中 说明/key', () => {
    expect(filterParams(params, {}, '欠费', '')).toHaveLength(1)
    expect(filterParams(params, {}, 'ttl', '')).toHaveLength(1)
    expect(filterParams(params, {}, '不存在', '')).toHaveLength(0)
  })
  it('状态筛选按草稿是否改动', () => {
    const draft = { 'arrears.threshold': '200' }
    expect(filterParams(params, draft, '', 'changed')).toEqual([params[0]])
    expect(filterParams(params, draft, '', 'origin')).toEqual([params[1]])
  })
})

const logs: AuditLog[] = [
  { logId: '1', time: '2026-08-17 10:00:00', operator: '张三', type: '权限变更', action: '调整菜单权限', ip: '10.0.0.1' },
  { logId: '2', time: '2026-08-16 09:00:00', operator: '李四', type: '状态变更', action: '订单取消', ip: '10.0.0.2' },
]

describe('filterAuditLogs', () => {
  it('关键字命中 操作人/内容', () => {
    expect(filterAuditLogs(logs, '张三', '', '')).toHaveLength(1)
    expect(filterAuditLogs(logs, '订单', '', '')).toHaveLength(1)
  })
  it('类型与日期前缀组合筛选', () => {
    expect(filterAuditLogs(logs, '', '状态变更', '')).toHaveLength(1)
    expect(filterAuditLogs(logs, '', '', '2026-08-16')).toHaveLength(1)
    expect(filterAuditLogs(logs, '张三', '状态变更', '')).toHaveLength(0)
  })
})
