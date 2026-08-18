// 子公司/法人页:列名以 fields.md 1.4 为准(法人编码/法人名称);原型多余列(品牌/区域/部门数)后端无字段,按契约裁剪。
// 契约: GET/POST /legal-entities、PUT /legal-entities/{id}(org.yaml)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { PageHead, pagerTexts } from '../shared'
import { filterLegalEntities, pageSlice, type LegalEntityRow } from './filter'
import '../org.css'

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
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.company.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn org-btn-primary"
            onClick={() => setForm({ id: 0, code: '', name: '' })}>{t.pages.company.add}</button>
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{t.pages.company.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.code}</td>
                    <td>{r.name}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setForm({ id: r.id, code: r.code, name: r.name })}>{t.pages.company.edit}</button>
                        <span className="sep">|</span>
                        <button onClick={() => setDetail(r)}>{t.pages.company.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={3}><div className="org-empty">{t.pages.company.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.company)} />
        </div>
      </div>
      {detail && (
        <Drawer title={t.pages.company.detail} onClose={() => setDetail(null)}
          footer={<button className="org-btn org-btn-primary" onClick={() => setDetail(null)}>{t.pages.company.cancel}</button>}>
          <div className="org-detail-list">
            <div className="org-detail-item"><span className="k">{t.pages.company.codeLabel}</span><span className="v">{detail.code}</span></div>
            <div className="org-detail-item"><span className="k">{t.pages.company.nameLabel}</span><span className="v">{detail.name}</span></div>
          </div>
        </Drawer>
      )}
      {form && (
        <Drawer title={form.id ? t.pages.company.edit : t.pages.company.add} onClose={() => setForm(null)}
          footer={
            <>
              <button className="org-btn" onClick={() => setForm(null)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy} onClick={submit}>{t.pages.company.save}</button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{t.pages.company.codeLabel}</label>
              <input className="org-input" value={form.code} placeholder={t.pages.company.codePlaceholder}
                onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </div>
            <div className="org-field">
              <label><span className="req">*</span>{t.pages.company.nameLabel}</label>
              <input className="org-input" value={form.name} placeholder={t.pages.company.namePlaceholder}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            {formError && <div className="org-error" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
