// 账号列表筛选用例。
import { describe, expect, it } from 'vitest'
import { filterAccounts, type AccountRow } from './list'

const rows: AccountRow[] = [
  { id: 1, username: 'ops_wang', realName: '王五', roleName: '运营', legalEntityName: 'A 公司', deptName: '', postName: '', regionScope: '', status: 1 },
  { id: 2, username: 'sys_admin', realName: '管理员', roleName: '系统管理员', legalEntityName: '', deptName: '', postName: '', regionScope: '', status: 0 },
]

describe('filterAccounts', () => {
  it('关键字命中 账号/姓名/角色名', () => {
    expect(filterAccounts(rows, 'wang', '', '')).toHaveLength(1)
    expect(filterAccounts(rows, '管理员', '', '')).toHaveLength(1)
    expect(filterAccounts(rows, '不存在', '', '')).toHaveLength(0)
  })
  it('角色名精确 + 状态筛选', () => {
    expect(filterAccounts(rows, '', '运营', '')).toHaveLength(1)
    expect(filterAccounts(rows, '', '', '0')).toHaveLength(1)
    expect(filterAccounts(rows, '', '运营', '0')).toHaveLength(0)
  })
})
