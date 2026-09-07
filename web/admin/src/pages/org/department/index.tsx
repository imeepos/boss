// 部门管理页:列名以 fields.md 1.4 为准(部门/所属子公司);原型职能/岗位数/队列列后端无字段,按契约裁剪。
// 契约: GET /departments(org.yaml;legalEntityId 过滤,本次全量)。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterDepartments, pageSlice, type DepartmentRow } from './filter'
import { DeptFormDrawer, emptyDeptForm, rowToDeptForm, type DeptFormValues } from './DeptForm'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'

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
      toast.success(t.pages.staff.saved)
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.department.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const confirmDialog = useConfirm()

  // 补齐与后端 DELETE /departments/:deptId 对齐的行内入口(占用挂靠时后端 40900 拒)。
  const del = async (r: DepartmentRow) => {
    if (!(await confirmDialog(t.pages.staff.delDeptConfirm.replace('{name}', r.name), { danger: true }))) return
    try {
      await apiFetch(`/departments/${r.id}`, { method: 'DELETE' })
      toast.success(t.pages.staff.delOk)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t.pages.department.saveFail)
    }
  }

  const filtered = useMemo(() => filterDepartments(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={t.pages.department.title} desc={t.pages.department.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.department.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <BatchImportEntry kind="department" onImported={load} />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setForm(emptyDeptForm())}>
            {t.pages.department.create}
          </button>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{t.pages.department.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.legalEntity}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{t.pages.department.detail}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setForm(rowToDeptForm(r))}>{t.pages.account.edit}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button className="text-[var(--color-danger)]" onClick={() => del(r)}>{t.pages.staff.del}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={3} loading={busy} text={t.pages.department.empty} />}
              </tbody>
            </table>
          </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
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
