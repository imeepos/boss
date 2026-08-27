// 用户列表列渲染兜底回归(2026-09 事故:接口缺 createdAt 时页面渲染字面量 undefined)。
import { describe, expect, it } from 'vitest'
import { createdAtCell, filterUsers, loginNameCell, pageSlice, type UserRow } from './filter'

const base: UserRow = { customerId: 1, name: '王先生', phone: '13800000000' }

describe('用户列表列渲染兜底', () => {
  it('登录名缺失/空串渲染占位符,有值原样', () => {
    expect(loginNameCell(base)).toBe('—')
    expect(loginNameCell({ ...base, loginName: '' })).toBe('—')
    expect(loginNameCell({ ...base, loginName: '13800000000' })).toBe('13800000000')
  })
  it('注册时间缺失渲染占位符,不出现字面量 undefined', () => {
    expect(createdAtCell(base)).toBe('—')
    expect(createdAtCell({ ...base, createdAt: undefined })).toBe('—')
    expect(createdAtCell({ ...base, createdAt: '' })).toBe('—')
  })
  it('注册时间本地时区格式化', () => {
    expect(createdAtCell({ ...base, createdAt: '2026-08-21T10:00:00' })).toBe('2026-08-21 10:00:00')
  })
})

describe('filterUsers / pageSlice', () => {
  const rows = [
    base,
    { customerId: 2, name: '李工程师', phone: '13800002001', loginName: 'li_gcs' },
  ]
  it('关键字命中 姓名/手机号/登录名', () => {
    expect(filterUsers(rows, '李')).toHaveLength(1)
    expect(filterUsers(rows, '13800000000')).toHaveLength(1)
    expect(filterUsers(rows, 'li_gcs')).toHaveLength(1)
    expect(filterUsers(rows, '不存在')).toHaveLength(0)
    expect(filterUsers(rows, '  ')).toHaveLength(2)
  })
  it('pageSlice 分页', () => {
    expect(pageSlice(rows, 1, 1)).toHaveLength(1)
    expect(pageSlice(rows, 2, 1)).toHaveLength(1)
    expect(pageSlice(rows, 3, 1)).toHaveLength(0)
  })
})
