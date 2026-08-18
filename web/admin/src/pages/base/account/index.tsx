// 账号与角色页(A1 定标样板):列名严格对照 docs/admin/account.html 原型。
// 账号/姓名/角色/子公司/部门/岗位/数据范围/状态/操作。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { StatusTag } from '../../../components/StatusTag'

interface AccountRow {
  id: number
  username: string
  realName: string
  roleName: string
  legalEntityName: string
  deptName: string
  postName: string
  regionScope: string
  status: number
}

const COLUMNS = ['账号', '姓名', '角色', '子公司', '部门', '岗位', '数据范围', '状态', '操作']

export default function AccountListPage() {
  const [rows, setRows] = useState<AccountRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')

  useEffect(() => {
    apiFetch<AccountRow[]>('/accounts')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [])

  const filtered = keyword
    ? rows.filter((r) => r.username.includes(keyword) || r.realName.includes(keyword))
    : rows

  return (
    <div>
      <h2>账号与角色</h2>
      <div style={{ marginBottom: 12 }}>
        <input
          placeholder="按账号/姓名筛选"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ padding: '6px 10px', width: 240 }}
        />
      </div>
      {error && <div style={{ color: '#e54545' }}>{error}</div>}
      <table style={{ width: '100%', background: '#fff', borderCollapse: 'collapse', fontSize: 13 }}>
        <thead>
          <tr>
            {COLUMNS.map((c) => (
              <th key={c} style={th}>{c}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {filtered.map((r) => (
            <tr key={r.id}>
              <td style={td}>{r.username}</td>
              <td style={td}>{r.realName}</td>
              <td style={td}>{r.roleName}</td>
              <td style={td}>{r.legalEntityName || '—'}</td>
              <td style={td}>{r.deptName || '—'}</td>
              <td style={td}>{r.postName || '—'}</td>
              <td style={td}>{r.regionScope || '全集团'}</td>
              <td style={td}>
                <StatusTag domain="accountStatus" value={String(r.status)} />
              </td>
              <td style={td}>
                <a style={{ color: '#1677ff', cursor: 'pointer' }}>编辑</a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <p style={{ color: '#888' }}>共 {filtered.length} 条</p>
    </div>
  )
}

const th: React.CSSProperties = {
  textAlign: 'left', padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #eee',
}
const td: React.CSSProperties = { padding: '8px 12px', borderBottom: '1px solid #f5f5f5' }
