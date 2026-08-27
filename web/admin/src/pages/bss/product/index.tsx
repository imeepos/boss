// 产品资费页:列名以 fields.md §2.2 为准;契约 GET/POST /products + PUT /products/{id}
// + PUT /products/{id}/status + POST /products/{id}/price-history(调价)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import type { ProductCategory, ProductRow, ProductStatus } from './types'
import { PRODUCT_CATEGORIES, PRODUCT_STATUSES } from './types'
import { fmtFee, fmtTime } from '../../../lib/format'
import { PriceHistoryDrawer } from './PriceHistoryDrawer'
import { PriceChangeDrawer } from './PriceChangeDrawer'
import { emptyProductForm, ProductFormDrawer, type ProductFormValues } from './ProductForm'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { TableStateRow } from '../../../components/business'

function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}

export default function ProductPage() {
  const t = useT()
  const p = t.pages.product
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<ProductRow[]>([])
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])
  const [company, setCompany] = useState(0)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<ProductRow | null>(null)
  const [history, setHistory] = useState<ProductRow | null>(null)
  const [priceTarget, setPriceTarget] = useState<ProductRow | null>(null)
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
      if (form.id > 0) {
        await apiFetch(`/products/${form.id}`, {
          method: 'PUT',
          body: { name: form.name.trim(), bandwidth: form.bandwidth.trim(), category: form.category },
        })
      } else {
        await apiFetch('/products', {
          method: 'POST',
          body: {
            legalEntityId: form.legalEntityId,
            name: form.name.trim(),
            bandwidth: form.bandwidth.trim(),
            monthlyFee: Number(form.monthlyFee),
            category: form.category,
            status: form.status,
          },
        })
      }
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : p.saveFail)
    } finally {
      setBusy(false)
    }
  }

  // 上下架:发布即生效;OFFLINE→PUBLISHED 重新上架同口径。
  const toggleStatus = async (r: ProductRow) => {
    const next = r.status === 'PUBLISHED' ? 'OFFLINE' : 'PUBLISHED'
    const msg = next === 'PUBLISHED' ? p.publishConfirm : p.unpublishConfirm
    if (!(await confirmDialog(msg, { title: next === 'PUBLISHED' ? p.publish : p.unpublish }))) return
    setError('')
    setBusy(true)
    try {
      await apiFetch(`/products/${r.id}/status`, { method: 'PUT', body: { status: next } })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.actFail)
      setBusy(false)
    }
  }

  const companyName = (id: number) => companies.find((c) => c.id === id)?.name ?? `#${id}`
  const toCategory = (raw: string): ProductCategory => PRODUCT_CATEGORIES.includes(raw as ProductCategory) ? raw as ProductCategory : 'broadband'
  const toStatus = (raw: string): ProductStatus => PRODUCT_STATUSES.includes(raw as ProductStatus) ? raw as ProductStatus : 'DRAFT'
  const categoryLabel = (raw: string) => p.categoryOptions[PRODUCT_CATEGORIES.indexOf(toCategory(raw))] ?? (raw || '—')
  const statusLabel = (raw: string) => p.statusOptions[PRODUCT_STATUSES.indexOf(toStatus(raw))] ?? raw
  const slice = useMemo(() => pageSlice(rows, page, pageSize), [rows, page, pageSize])

  const act = 'cursor-pointer border-none bg-transparent p-0 text-[13px] text-[var(--color-text-link)] hover:underline'
  const sep = <span className="text-[var(--shell-side-border)]">|</span>

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
          <BatchImportEntry kind="product" onImported={load} />
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
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{categoryLabel(r.category)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.bandwidth || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.monthlyFee)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.effectiveAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="product" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center gap-2">
                        <button className={act} onClick={() => setDetail(r)}>{p.detail}</button>
                        {sep}
                        <button className={act} onClick={() => setForm({
                          id: r.id, legalEntityId: r.legalEntityId, name: r.name,
                          bandwidth: r.bandwidth || '', category: toCategory(r.category),
                          monthlyFee: String(r.monthlyFee), status: toStatus(r.status),
                        })}>{p.edit}</button>
                        {sep}
                        <button className={act} onClick={() => setPriceTarget(r)}>{p.priceChange}</button>
                        {sep}
                        <button className={act} disabled={busy} onClick={() => toggleStatus(r)}>{r.status === 'PUBLISHED' ? p.unpublish : p.publish}</button>
                        {sep}
                        <button className={act} onClick={() => setHistory(r)}>{p.history}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={8} loading={busy} text={p.empty} />}
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
            { k: p.columns[2], v: categoryLabel(detail.category) },
            { k: p.columns[3], v: detail.bandwidth },
            { k: p.columns[4], v: fmtFee(detail.monthlyFee) },
            { k: p.columns[5], v: fmtTime(detail.effectiveAt) },
            { k: p.columns[6], v: statusLabel(detail.status) },
            { k: 'ID', v: String(detail.id) },
          ]}
        />
      )}
      {history && (
        <PriceHistoryDrawer productId={history.id} productName={history.name} onClose={() => setHistory(null)} />
      )}
      {priceTarget && (
        <PriceChangeDrawer productId={priceTarget.id} productName={priceTarget.name} currentFee={priceTarget.monthlyFee}
          onClose={() => setPriceTarget(null)} onDone={() => { setPriceTarget(null); load() }} />
      )}
      <ProductFormDrawer
        open={form !== null}
        values={form ?? emptyProductForm()}
        companyName={form && form.id > 0 ? companyName(form.legalEntityId) : undefined}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />
    </div>
  )
}
