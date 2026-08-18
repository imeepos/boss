// 产品资费页:列名以 fields.md §2.2 为准;契约 GET /products(legalEntityId 过滤)+ POST /products。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import type { ProductRow } from './types'
import { fmtTime } from '../../../lib/format'
import { fmtFee, PriceHistoryDrawer } from './PriceHistoryDrawer'
import { emptyProductForm, ProductFormDrawer, type ProductFormValues } from './ProductForm'
import '../../org/org.css'

function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}

export default function ProductPage() {
  const t = useT()
  const p = t.pages.product
  const [rows, setRows] = useState<ProductRow[]>([])
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])
  const [company, setCompany] = useState(0)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<ProductRow | null>(null)
  const [history, setHistory] = useState<ProductRow | null>(null)
  const [form, setForm] = useState<ProductFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ProductRow[] }>('/products', { query: { legalEntityId: company || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    apiFetch<{ id: number; name: string }[]>('/legal-entities')
      .then((d) => setCompanies(d ?? []))
      .catch(() => setCompanies([]))
  }, [])
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (!form || busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/products', {
        method: 'POST',
        body: {
          legalEntityId: form.legalEntityId,
          name: form.name.trim(),
          bandwidth: form.bandwidth.trim(),
          monthlyFee: Number(form.monthlyFee),
          status: form.status,
        },
      })
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : p.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const companyName = (id: number) => companies.find((c) => c.id === id)?.name ?? `#${id}`
  const slice = useMemo(() => pageSlice(rows, page, pageSize), [rows, page, pageSize])

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <select className="org-select" value={company ? String(company) : ''}
            onChange={(e) => { setCompany(Number(e.target.value) || 0); setPage(1); load() }}>
            <option value="">{p.allCompany}</option>
            {companies.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setForm(emptyProductForm())}>{p.create}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{p.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{companyName(r.legalEntityId)}</td>
                    <td>{r.name}</td>
                    <td>{r.bandwidth || '—'}</td>
                    <td>{fmtFee(r.monthlyFee)}</td>
                    <td>{fmtTime(r.effectiveAt)}</td>
                    <td><StatusTag domain="product" value={r.status} /></td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{p.detail}</button>
                        <span className="sep">|</span>
                        <button onClick={() => setHistory(r)}>{p.history}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{p.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
      {detail && (
        <DetailDrawer
          title={p.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: p.columns[0], v: companyName(detail.legalEntityId) },
            { k: p.columns[1], v: detail.name },
            { k: p.columns[2], v: detail.bandwidth },
            { k: p.columns[3], v: fmtFee(detail.monthlyFee) },
            { k: p.columns[4], v: fmtTime(detail.effectiveAt) },
            { k: p.columns[5], v: detail.status },
            { k: 'ID', v: String(detail.id) },
          ]}
        />
      )}
      {history && (
        <PriceHistoryDrawer productId={history.id} productName={history.name} onClose={() => setHistory(null)} />
      )}
      <ProductFormDrawer
        open={form !== null}
        values={form ?? emptyProductForm()}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />
    </div>
  )
}
