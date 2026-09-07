// 账号与角色页:Pro 惯例列表(筛选/分页/状态列/行操作) + 新建/编辑 Drawer 表单。
// 契约: GET/POST /accounts、PUT /accounts/{id}(sys.yaml;列名对照 docs/admin/account.html)。
// 停用确认走 ConfirmDialog;DELETE /accounts/{id} 经 102 实测为软删(等价停用),
// 不另设重复删除入口(结论见 docs/acceptance/2026-09-05-pp1a-page-polish.md 清单5)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { toast } from 'sonner'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { useConfirm } from '../../../components/ConfirmDialog'
import { DataTable } from '../../../components/business/data-table'
import { ActionLinks, ActionLink, ActionSep } from '../../../components/business/page-head'
import { DetailDrawer } from '../../org/shared'
import { filterAccounts, pageSlice, type AccountRow } from './list'
import { buildAccountPayload, validateAccount, type AccountFormValues } from './form'
import { AccountFormDrawer } from './AccountForm'
import { BatchImportEntry } from '../importer/BatchImportEntry'

export default function AccountListPage() {
  const t = useT()
  const confirmDialog = useConfirm()
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
      toast.success(t.pages.account.saved)
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.account.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const disable = async (row: AccountRow) => {
    if (busy) return
    if (!(await confirmDialog(t.pages.account.disableConfirm.replace('{name}', row.username), { danger: true }))) return
    setBusy(true)
    try {
      // PUT 为全量语义(域层校验 username/realName/roleCode):禁用=全字段 + status 翻转。
      const f = rowToForm(row)
      await apiFetch(`/accounts/${row.id}`, {
        method: 'PUT',
        body: {
          ...buildAccountPayload(f, true),
          status: row.status === 1 ? 0 : 1,
        },
      })
      toast.success(t.pages.account.statusUpdated)
      load()
    } catch (e) {
      // 行内停用失败:抽屉未开,formError 不可见,必须走 toast。
      toast.error(e instanceof Error ? e.message : t.pages.account.saveFail)
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
          <Dropdown
            value={role}
            options={[{ value: '', label: t.pages.account.allRole }, ...roles.map((r) => ({ value: r, label: r }))]}
            onChange={(v) => { setRole(v); setPage(1) }}
            ariaLabel={t.pages.account.allRole}
          />
          <Dropdown
            value={status}
            options={[
              { value: '', label: t.pages.account.allStatus },
              { value: '1', label: t.pages.account.statusOn },
              { value: '0', label: t.pages.account.statusOff },
            ]}
            onChange={(v) => { setStatus(v); setPage(1) }}
            ariaLabel={t.pages.account.allStatus}
          />
          <span className="spacer" />
          <BatchImportEntry kind="account" onImported={load} />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setForm(emptyForm())}>
            {t.pages.account.create}
          </button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="px-4 pb-4">
            <DataTable
              emptyText={t.pages.account.empty}
              rows={slice as unknown as Record<string, unknown>[]}
              columns={[
                { key: 'username', label: t.pages.account.columns[0] },
                { key: 'realName', label: t.pages.account.columns[1] },
                { key: 'roleName', label: t.pages.account.columns[2] },
                { key: 'legalEntityName', label: t.pages.account.columns[3], render: (r) => String((r as unknown as AccountRow).legalEntityName || '—') },
                { key: 'deptName', label: t.pages.account.columns[4], render: (r) => String((r as unknown as AccountRow).deptName || '—') },
                { key: 'postName', label: t.pages.account.columns[5], render: (r) => String((r as unknown as AccountRow).postName || '—') },
                { key: 'scope', label: t.pages.account.columns[6], render: (r) => scopeText(r as unknown as AccountRow) },
                { key: 'status', label: t.pages.account.columns[7], render: (r) => <StatusTag domain="accountStatus" value={String((r as unknown as AccountRow).status)} /> },
                { key: 'op', label: t.pages.account.columns[8], render: (r) => { const row = r as unknown as AccountRow; return (
                  <ActionLinks>
                    <ActionLink label={t.pages.account.detail} onClick={() => setDetail(row)} />
                    <ActionSep />
                    <ActionLink label={t.pages.account.edit} onClick={() => setForm(rowToForm(row))} />
                    {row.status === 1 && (
                      <>
                        <ActionSep />
                        <ActionLink label={t.pages.account.disable} onClick={() => void disable(row)} />
                      </>
                    )}
                  </ActionLinks>
                ) } },
              ]}
            />
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
