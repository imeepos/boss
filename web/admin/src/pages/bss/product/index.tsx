// 产品资费页:列名以 fields.md §2.2 为准;契约 GET /products(legalEntityId 过滤)+ POST /products。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import type { ProductRow } from './types'
import { fmtFee, fmtTime } from '../../../lib/format'
import { PriceHistoryDrawer } from './PriceHistoryDrawer'
import { emptyProductForm, ProductFormDrawer, type ProductFormValues } from './ProductForm'
import { TableStateRow } from '../../../components/business'

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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={company ? String(company) : ''}
            options={[{ value: '', label: p.allCompany }, ...companies.map((c) => ({ value: String(c.id), label: c.name }))]}
            onChange={(v) => { setCompany(Number(v) || 0); setPage(1); load() }}
            ariaLabel={p.allCompany}
          />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setForm(emptyProductForm())}>{p.create}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{p.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{companyName(r.legalEntityId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.bandwidth || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.monthlyFee)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.effectiveAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="product" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{p.detail}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setHistory(r)}>{p.history}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={p.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
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
