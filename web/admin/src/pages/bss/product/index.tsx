// 产品资费页:列名以 fields.md §2.2 为准;契约 GET/POST /products + PUT /products/{id}
// + PUT /products/{id}/status + POST /products/{id}/price-history(调价)。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { ActionLink, ActionLinks, ActionSep, ErrorBanner, TableStateRow, ToolbarButton } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import type { ProductCategory, ProductRow, ProductStatus } from './types'
import { PRODUCT_CATEGORIES, PRODUCT_STATUSES } from './types'
import { fmtFee, fmtTime } from '../../../lib/format'
import { PriceHistoryDrawer } from './PriceHistoryDrawer'
import { PriceChangeDrawer } from './PriceChangeDrawer'
import { emptyProductForm, ProductFormDrawer, type ProductFormValues } from './ProductForm'
import { BindTemplateDrawer } from './BindTemplateDrawer'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import type { OfferBindingRow } from './types'

function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}

export default function ProductPage() {
  const t = useT()
  const p = t.pages.product
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<ProductRow[]>([])
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])
  const [bindings, setBindings] = useState<Map<number, OfferBindingRow>>(new Map())
  const [company, setCompany] = useState(0)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<ProductRow | null>(null)
  const [history, setHistory] = useState<ProductRow | null>(null)
  const [priceTarget, setPriceTarget] = useState<ProductRow | null>(null)
  const [bindTarget, setBindTarget] = useState<ProductRow | null>(null)
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
  const loadBindings = () => {
    apiFetch<{ items: OfferBindingRow[] }>('/provision-bindings')
      .then((d) => setBindings(new Map((d?.items ?? []).map((b) => [b.offerId, b]))))
      .catch((e) => {
        // 模板绑定列取不到一律显示"未绑定",需留痕便于发现接口异常
        setBindings(new Map())
        console.warn('[product] provision-bindings 拉取失败:', e instanceof Error ? e.message : e)
      })
  }
  useEffect(() => {
    apiFetch<{ id: number; name: string }[]>('/legal-entities')
      .then((d) => setCompanies(d ?? []))
      .catch((e) => {
        setCompanies([])
        console.warn('[product] legal-entities 拉取失败:', e instanceof Error ? e.message : e)
      })
    loadBindings()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
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
      toast.success(p.savedOk)
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
    if (busy) return
    const next = r.status === 'PUBLISHED' ? 'OFFLINE' : 'PUBLISHED'
    const msg = next === 'PUBLISHED' ? p.publishConfirm : p.unpublishConfirm
    if (!(await confirmDialog(msg, { title: next === 'PUBLISHED' ? p.publish : p.unpublish }))) return
    setError('')
    setBusy(true)
    try {
      await apiFetch(`/products/${r.id}/status`, { method: 'PUT', body: { status: next } })
      toast.success(next === 'PUBLISHED' ? p.publishOk : p.unpublishOk)
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

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={company ? String(company) : ''}
            options={[{ value: '', label: p.allCompany }, ...companies.map((c) => ({ value: String(c.id), label: c.name }))]}
            onChange={(v) => { setCompany(Number(v) || 0); setPage(1); load() }}
            ariaLabel={p.allCompany}
          />
          <span className="spacer" />
          <BatchImportEntry kind="product" onImported={load} />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          <ToolbarButton primary onClick={() => setForm(emptyProductForm())}>{p.create}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="overflow-x-auto px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{p.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>{companyName(r.legalEntityId)}</TableCell>
                    <TableCell>{r.name}</TableCell>
                    <TableCell>{categoryLabel(r.category)}</TableCell>
                    <TableCell>{r.bandwidth || '—'}</TableCell>
                    <TableCell>{fmtFee(r.monthlyFee)}</TableCell>
                    <TableCell>{fmtTime(r.effectiveAt)}</TableCell>
                    <TableCell><StatusTag domain="product" value={r.status} /></TableCell>
                    <TableCell>
                      {(() => {
                        const b = bindings.get(r.id)
                        return b ? `${b.templateName} (${b.templateCode})` : <span className="text-[var(--shell-group-title)]">{p.templateUnbound}</span>
                      })()}
                    </TableCell>
                    <TableCell>
                      <ActionLinks>
                        <ActionLink onClick={() => setDetail(r)} label={p.detail} testId={'prod-detail-' + r.id} />
                        <ActionSep />
                        <ActionLink onClick={() => setForm({
                          id: r.id, legalEntityId: r.legalEntityId, name: r.name,
                          bandwidth: r.bandwidth || '', category: toCategory(r.category),
                          monthlyFee: String(r.monthlyFee), status: toStatus(r.status),
                        })} label={p.edit} />
                        <ActionSep />
                        <ActionLink onClick={() => setPriceTarget(r)} label={p.priceChange} />
                        <ActionSep />
                        <ActionLink onClick={() => setBindTarget(r)} label={p.templateBind} />
                        <ActionSep />
                        <ActionLink onClick={() => toggleStatus(r)} label={r.status === 'PUBLISHED' ? p.unpublish : p.publish} testId={'prod-toggle-' + r.id} />
                        <ActionSep />
                        <ActionLink onClick={() => setHistory(r)} label={p.history} />
                      </ActionLinks>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={9} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </Card>
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
      {bindTarget && (
        <BindTemplateDrawer offerId={bindTarget.id} offerName={bindTarget.name} legalEntityId={bindTarget.legalEntityId}
          onClose={() => setBindTarget(null)} onDone={() => { setBindTarget(null); loadBindings() }} />
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
