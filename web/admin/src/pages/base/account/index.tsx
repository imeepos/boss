// 账号与角色页:Pro 惯例列表(筛选/分页/状态列/行操作) + 新建/编辑 Drawer 表单。
// 契约: GET/POST /accounts、PUT /accounts/{id}(sys.yaml;列名对照 docs/admin/account.html)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { DetailDrawer } from '../../org/shared'
import { filterAccounts, pageSlice, type AccountRow } from './list'
import { buildAccountPayload, validateAccount, type AccountFormValues } from './form'
import { AccountFormDrawer } from './AccountForm'
import '../../org/org.css'
import './account.css'

export default function AccountListPage() {
  const t = useT()
  const [rows, setRows] = useState<AccountRow[]>([])
  const [roles, setRoles] = useState<string[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [role, setRole] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<AccountRow | null>(null)
  const [form, setForm] = useState<AccountFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)
  const [confirmDisable, setConfirmDisable] = useState<AccountRow | null>(null)

  const load = () => {
    setError('')
    apiFetch<AccountRow[]>('/accounts')
      .then((d) => {
        setRows(d ?? [])
        setRoles([...new Set((d ?? []).map((r) => r.roleName))])
      })
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.account.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => filterAccounts(rows, keyword, role, status), [rows, keyword, role, status])
  const slice = pageSlice(filtered, page, pageSize)

  const submit = async () => {
    if (!form || busy) return
    const errs = validateAccount(form, Boolean(form.id))
    if (errs.length) return
    setBusy(true)
    setFormError('')
    try {
      const body = buildAccountPayload(form, Boolean(form.id))
      if (form.id) await apiFetch(`/accounts/${form.id}`, { method: 'PUT', body })
      else await apiFetch('/accounts', { method: 'POST', body })
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.account.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const disable = async () => {
    if (!confirmDisable || busy) return
    setBusy(true)
    try {
      // PUT 为全量语义(域层校验 username/realName/roleCode):禁用=全字段 + status 翻转。
      const f = rowToForm(confirmDisable)
      await apiFetch(`/accounts/${confirmDisable.id}`, {
        method: 'PUT',
        body: {
          ...buildAccountPayload(f, true),
          status: confirmDisable.status === 1 ? 0 : 1,
        },
      })
      setConfirmDisable(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.account.saveFail)
      setConfirmDisable(null)
    } finally {
      setBusy(false)
    }
  }

  const scopeText = (r: AccountRow) => r.regionScope || t.pages.account.scopeAll

  return (
    <div>
      <div className="org-page-head">
        <h2 className="org-page-title">{t.pages.account.title}</h2>
        <p className="org-page-desc">{t.pages.account.desc}</p>
      </div>
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.account.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <select className="org-select" value={role} onChange={(e) => { setRole(e.target.value); setPage(1) }}>
            <option value="">{t.pages.account.allRole}</option>
            {roles.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <select className="org-select" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{t.pages.account.allStatus}</option>
            <option value="1">{t.pages.account.statusOn}</option>
            <option value="0">{t.pages.account.statusOff}</option>
          </select>
          <span className="spacer" />
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setForm(emptyForm())}>
            {t.pages.account.create}
          </button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{t.pages.account.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.username}</td>
                    <td>{r.realName}</td>
                    <td>{r.roleName}</td>
                    <td>{r.legalEntityName || '—'}</td>
                    <td>{r.deptName || '—'}</td>
                    <td>{r.postName || '—'}</td>
                    <td>{scopeText(r)}</td>
                    <td><StatusTag domain="accountStatus" value={String(r.status)} /></td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{t.pages.account.detail}</button>
                        <span className="sep">|</span>
                        <button onClick={() => setForm(rowToForm(r))}>{t.pages.account.edit}</button>
                        {r.status === 1 && (
                          <>
                            <span className="sep">|</span>
                            <button onClick={() => setConfirmDisable(r)}>{t.pages.account.disable}</button>
                          </>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={9}><div className="org-empty">{t.pages.account.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize}
            rangeText={t.pages.company.rangeText} prevText={t.pages.company.prev}
            nextText={t.pages.company.next} perPageText={t.pages.company.perPage}
            jumpText={t.pages.company.jumpText} pageUnitText={t.pages.company.pageUnit} />
        </div>
      </div>

      {detail && (
        <DetailDrawer title={t.pages.account.detailTitle} closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.account.columns[0], v: detail.username },
            { k: t.pages.account.columns[1], v: detail.realName },
            { k: t.pages.account.columns[2], v: detail.roleName },
            { k: t.pages.account.columns[3], v: detail.legalEntityName },
            { k: t.pages.account.columns[4], v: detail.deptName },
            { k: t.pages.account.columns[5], v: detail.postName },
            { k: t.pages.account.columns[6], v: scopeText(detail) },
            { k: t.pages.account.fPhone, v: detail.phone || '—' },
          ]}
        />
      )}

      <AccountFormDrawer
        open={form !== null}
        values={form ?? emptyForm()}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />

      {confirmDisable && (
        <div className="acc-confirm-mask" onClick={() => setConfirmDisable(null)}>
          <div className="acc-confirm" onClick={(e) => e.stopPropagation()}>
            <p>{t.pages.account.disableConfirm.replace('{name}', confirmDisable.username)}</p>
            <div className="acc-confirm-actions">
              <button className="org-btn" onClick={() => setConfirmDisable(null)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy} onClick={disable}>{t.pages.account.disable}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function emptyForm(): AccountFormValues {
  return { username: '', password: '', realName: '', phone: '', roleCode: '', legalEntityId: 0, deptId: 0, postId: 0, regionScope: '' }
}

function rowToForm(r: AccountRow): AccountFormValues {
  return {
    id: r.id,
    username: r.username,
    password: '',
    realName: r.realName,
    phone: r.phone ?? '',
    roleCode: r.roleCode ?? '',
    legalEntityId: r.legalEntityId ?? 0,
    deptId: r.deptId ?? 0,
    postId: r.postId ?? 0,
    regionScope: r.regionScope ?? '',
  }
}
