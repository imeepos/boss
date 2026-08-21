// 子公司/法人页:列名以 fields.md 1.4 为准(法人编码/法人名称);原型多余列(品牌/区域/部门数)后端无字段,按契约裁剪。
// 契约: GET/POST /legal-entities、PUT /legal-entities/{id}(org.yaml)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { PageHead, pagerTexts } from '../shared'
import { filterLegalEntities, pageSlice, type LegalEntityRow } from './filter'
import { TableStateRow } from '../../../components/business'

export default function CompanyPage() {
  const t = useT()
  const [rows, setRows] = useState<LegalEntityRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<LegalEntityRow | null>(null)
  const [form, setForm] = useState<{ id: number; code: string; name: string } | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    apiFetch<LegalEntityRow[]>('/legal-entities')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.company.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => filterLegalEntities(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  const submit = async () => {
    if (!form || busy) return
    if (!form.code.trim() || !form.name.trim()) { setFormError(t.pages.company.requiredHint); return }
    setBusy(true)
    setFormError('')
    try {
      const body = { code: form.code.trim(), name: form.name.trim() }
      if (form.id) await apiFetch(`/legal-entities/${form.id}`, { method: 'PUT', body })
      else await apiFetch('/legal-entities', { method: 'POST', body })
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.company.saveFail)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <PageHead title={t.pages.company.title} desc={t.pages.company.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.company.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
            onClick={() => setForm({ id: 0, code: '', name: '' })}>{t.pages.company.add}</button>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{t.pages.company.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.code}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setForm({ id: r.id, code: r.code, name: r.name })}>{t.pages.company.edit}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setDetail(r)}>{t.pages.company.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={3} loading={busy} text={t.pages.company.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.company)} />
        </div>
      </div>
      {detail && (
        <Drawer title={t.pages.company.detail} onClose={() => setDetail(null)}
          footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setDetail(null)}>{t.pages.company.cancel}</button>}>
          <div className="flex flex-col gap-2.5">
            <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{t.pages.company.codeLabel}</span><span className="break-all text-[var(--shell-content-text)]">{detail.code}</span></div>
            <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{t.pages.company.nameLabel}</span><span className="break-all text-[var(--shell-content-text)]">{detail.name}</span></div>
          </div>
        </Drawer>
      )}
      {form && (
        <Drawer title={form.id ? t.pages.company.edit : t.pages.company.add} onClose={() => setForm(null)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setForm(null)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>{t.pages.company.save}</button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.company.codeLabel}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={form.code} placeholder={t.pages.company.codePlaceholder}
                onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.company.nameLabel}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={form.name} placeholder={t.pages.company.namePlaceholder}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            {formError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
