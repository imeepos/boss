// 账号表单校验/负载组装用例。
import { describe, expect, it } from 'vitest'
import { buildAccountPayload, cascadeReset, validateAccount, type AccountFormValues } from './form'

const base: AccountFormValues = {
  username: 'ops_user1',
  password: 'secret1',
  realName: '张三',
  phone: '13800001234',
  roleCode: 'ops',
  legalEntityId: 2,
  deptId: 3,
  postId: 0,
  regionScope: 'root.luzon',
}

describe('validateAccount', () => {
  it('完整合法值通过', () => {
    expect(validateAccount(base, false)).toEqual([])
  })
  it('用户名规则 3-64 位字母/数字/_-.', () => {
    expect(validateAccount({ ...base, username: 'ab' }, false)).toContain('invalidUsername')
    expect(validateAccount({ ...base, username: '非法名' }, false)).toContain('invalidUsername')
  })
  it('新建密码必填,编辑留空=不改', () => {
    expect(validateAccount({ ...base, password: '' }, false)).toContain('shortPassword')
    expect(validateAccount({ ...base, password: '' }, true)).not.toContain('shortPassword')
    expect(validateAccount({ ...base, password: '123' }, true)).toContain('shortPassword')
  })
  it('角色必选,手机号格式可选校验', () => {
    expect(validateAccount({ ...base, roleCode: '' }, false)).toContain('roleRequired')
    expect(validateAccount({ ...base, phone: 'abc' }, false)).toContain('invalidPhone')
  })
})

describe('buildAccountPayload', () => {
  it('0/空 转 null,新建带密码', () => {
    const p = buildAccountPayload(base, false)
    expect(p.postId).toBeNull()
    expect(p.regionScope).toBe('root.luzon')
    expect(p.password).toBe('secret1')
  })
  it('编辑且密码留空不带 password 字段', () => {
    const p = buildAccountPayload({ ...base, password: '', regionScope: '' }, true)
    expect(p).not.toHaveProperty('password')
    expect(p.regionScope).toBeNull()
  })
})

describe('cascadeReset', () => {
  it('换公司清部门与岗位,换部门清岗位', () => {
    const r1 = cascadeReset(base, 'legalEntityId')
    expect(r1.deptId).toBe(0)
    expect(r1.postId).toBe(0)
    const r2 = cascadeReset({ ...base, postId: 9 }, 'deptId')
    expect(r2.postId).toBe(0)
    expect(r2.deptId).toBe(3)
  })
})
