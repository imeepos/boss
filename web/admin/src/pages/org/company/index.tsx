// 子公司/法人页:列名以 fields.md 1.4 为准(法人编码/法人名称);原型多余列(品牌/区域/部门数)后端无字段,按契约裁剪。
// 契约: GET/POST /legal-entities、PUT /legal-entities/{id}(org.yaml)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { PageHead, pagerTexts } from '../shared'
import { filterLegalEntities, pageSlice, type LegalEntityRow } from './filter'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { toast } from 'sonner'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { Input } from '../../../components/ui/input'
import { EntityStaffPanel } from './EntityStaffPanel'
import { TableStateRow } from '../../../components/business'

export default function CompanyPage() {
  const t = useT()
  const [rows, setRows] = useState<LegalEntityRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<LegalEntityRow | null>(null)
  const [form, setForm] = useState<{
    id: number; code: string; name: string; taxJurisdiction: string; taxChannel: string
  } | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)
  const [staffEntity, setStaffEntity] = useState<LegalEntityRow | null>(null)

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
      const body = {
        code: form.code.trim(), name: form.name.trim(),
        taxJurisdiction: form.taxJurisdiction, taxChannel: form.taxChannel,
      }
      if (form.id) await apiFetch(`/legal-entities/${form.id}`, { method: 'PUT', body })
      else await apiFetch('/legal-entities', { method: 'POST', body })
      toast.success(t.pages.company.saved)
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
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" placeholder={t.pages.company.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <BatchImportEntry kind="legal_entity" onImported={load} />
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
            onClick={() => setForm({ id: 0, code: '', name: '', taxJurisdiction: '', taxChannel: 'manual' })}>{t.pages.company.add}</button>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error && <ErrorBanner message={error} />}
        <Table>
          <TableHeader>
            <TableRow>
              {t.pages.company.columns.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {slice.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{r.code}</TableCell>
                <TableCell>{r.name}</TableCell>
                <TableCell>{r.taxJurisdiction === 'CN' ? t.pages.company.taxCN : r.taxJurisdiction === 'PH' ? t.pages.company.taxPH : t.pages.company.taxUndetermined}</TableCell>
                <TableCell>{r.taxChannel === 'leqi' ? t.pages.company.channelLeqi : r.taxChannel === 'bir_eis' ? t.pages.company.channelBIR : t.pages.company.channelManual}</TableCell>
                <TableCell>
                  <span className="inline-flex items-center gap-1.5">
                    <button className="cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => setStaffEntity(r)}>{t.pages.company.staff.action}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button className="cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => setForm({ id: r.id, code: r.code, name: r.name, taxJurisdiction: r.taxJurisdiction ?? '', taxChannel: r.taxChannel ?? 'manual' })}>{t.pages.company.edit}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button className="cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => setDetail(r)}>{t.pages.company.detail}</button>
                  </span>
                </TableCell>
              </TableRow>
            ))}
            {!slice.length && <TableStateRow colSpan={5} loading={busy} text={t.pages.company.empty} />}
          </TableBody>
        </Table>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.company)} />
        </div>
      </Card>
      {detail && (
        <Drawer title={t.pages.company.detail} onClose={() => setDetail(null)}
          footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setDetail(null)}>{t.pages.company.cancel}</button>}>
          <div className="flex flex-col gap-2.5">
            <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{t.pages.company.codeLabel}</span><span className="break-all text-[var(--shell-content-text)]">{detail.code}</span></div>
            <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{t.pages.company.nameLabel}</span><span className="break-all text-[var(--shell-content-text)]">{detail.name}</span></div>
            <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{t.pages.company.taxLabel}</span><span className="break-all text-[var(--shell-content-text)]">{detail.taxJurisdiction === 'CN' ? t.pages.company.taxCN : detail.taxJurisdiction === 'PH' ? t.pages.company.taxPH : t.pages.company.taxUndetermined}</span></div>
            <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{t.pages.company.channelLabel}</span><span className="break-all text-[var(--shell-content-text)]">{detail.taxChannel === 'leqi' ? t.pages.company.channelLeqi : detail.taxChannel === 'bir_eis' ? t.pages.company.channelBIR : t.pages.company.channelManual}</span></div>
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
              <Input value={form.code} placeholder={t.pages.company.codePlaceholder}
                onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.company.nameLabel}</label>
              <Input value={form.name} placeholder={t.pages.company.namePlaceholder}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{t.pages.company.taxLabel}</label>
              <SimplePicker value={form.taxJurisdiction}
                onChange={(v) => setForm({ ...form, taxJurisdiction: v })}
                options={[
                  { value: 'CN', label: t.pages.company.taxCN },
                  { value: 'PH', label: t.pages.company.taxPH },
                ]}
                emptyLabel={t.pages.company.taxUndetermined}
                ariaLabel={t.pages.company.taxLabel} />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{t.pages.company.channelLabel}</label>
              <SimplePicker value={form.taxChannel}
                onChange={(v) => setForm({ ...form, taxChannel: v })}
                options={[
                  { value: 'manual', label: t.pages.company.channelManual },
                  { value: 'leqi', label: t.pages.company.channelLeqi },
                  { value: 'bir_eis', label: t.pages.company.channelBIR },
                ]}
                ariaLabel={t.pages.company.channelLabel} />
            </div>
            {formError && <ErrorBanner message={formError} />}
          </div>
        </Drawer>
      )}
      {staffEntity && (
        <EntityStaffPanel entityId={staffEntity.id} entityName={staffEntity.name} onClose={() => setStaffEntity(null)} />
      )}
    </div>
  )
}
