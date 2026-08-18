// 部门管理页:列名以 fields.md 1.4 为准(部门/所属子公司);原型职能/岗位数/队列列后端无字段,按契约裁剪。
// 契约: GET /departments(org.yaml;legalEntityId 过滤,本次全量)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterDepartments, pageSlice, type DepartmentRow } from './filter'
import { DeptFormDrawer, emptyDeptForm, rowToDeptForm, type DeptFormValues } from './DeptForm'
import { Pagination } from '../../../components/Pagination'
import '../org.css'

export default function DepartmentPage() {
  const t = useT()
  const [rows, setRows] = useState<DepartmentRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<DepartmentRow | null>(null)
  const [form, setForm] = useState<DeptFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    apiFetch<DepartmentRow[]>('/departments')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.department.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (!form || busy) return
    setBusy(true)
    setFormError('')
    try {
      const body = { legalEntityId: form.legalEntityId, name: form.name.trim() }
      if (form.id) await apiFetch(`/departments/${form.id}`, { method: 'PUT', body })
      else await apiFetch('/departments', { method: 'POST', body })
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.department.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const filtered = useMemo(() => filterDepartments(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={t.pages.department.title} desc={t.pages.department.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.department.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setForm(emptyDeptForm())}>
            {t.pages.department.create}
          </button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{t.pages.department.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.name}</td>
                    <td>{r.legalEntity}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{t.pages.department.detail}</button>
                        <span className="sep">|</span>
                        <button onClick={() => setForm(rowToDeptForm(r))}>{t.pages.account.edit}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={3}><div className="org-empty">{t.pages.department.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.department)} />
        </div>
      </div>
      {detail && (
        <DetailDrawer
          title={t.pages.department.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.department.columns[0], v: detail.name },
            { k: t.pages.department.columns[1], v: detail.legalEntity },
            { k: 'ID', v: String(detail.id) },
          ]}
        />
      )}
      <DeptFormDrawer
        open={form !== null}
        values={form ?? emptyDeptForm()}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />
    </div>
  )
}
