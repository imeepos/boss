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
      <div className="mb-4">
        <h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{t.pages.account.title}</h2>
        <p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{t.pages.account.desc}</p>
      </div>
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.account.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={role} onChange={(e) => { setRole(e.target.value); setPage(1) }}>
            <option value="">{t.pages.account.allRole}</option>
            {roles.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{t.pages.account.allStatus}</option>
            <option value="1">{t.pages.account.statusOn}</option>
            <option value="0">{t.pages.account.statusOff}</option>
          </select>
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setForm(emptyForm())}>
            {t.pages.account.create}
          </button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{t.pages.account.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.username}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.realName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.roleName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.legalEntityName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.deptName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.postName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{scopeText(r)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="accountStatus" value={String(r.status)} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{t.pages.account.detail}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setForm(rowToForm(r))}>{t.pages.account.edit}</button>
                        {r.status === 1 && (
                          <>
                            <span className="text-[var(--shell-side-border)]">|</span>
                            <button onClick={() => setConfirmDisable(r)}>{t.pages.account.disable}</button>
                          </>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={9} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.account.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
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
        <div className="fixed inset-0 z-[120] flex items-center justify-center bg-black/45" onClick={() => setConfirmDisable(null)}>
          <div className="w-90 rounded-md bg-[var(--shell-card-bg)] p-5" onClick={(e) => e.stopPropagation()}>
            <p className="m-0 mb-4 text-sm text-[var(--shell-content-text)]">{t.pages.account.disableConfirm.replace('{name}', confirmDisable.username)}</p>
            <div className="flex justify-end gap-2">
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setConfirmDisable(null)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={disable}>{t.pages.account.disable}</button>
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
