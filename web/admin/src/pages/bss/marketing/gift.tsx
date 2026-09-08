// 赠送时长阶梯规则 Tab:列表 + 抽屉式新建 + 停用(6送1/12送3/24送6 类规则)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { listGiftRules, createGiftRule, disableGiftRule, type GiftRule } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { RuleStatus } from './RuleStatus'
import {
  PageHead, pagerTexts, ErrorBanner, ToolbarButton, FormField, ActionLink,
  DataTable, type ColumnDef,
} from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Input } from '../../../components/ui/input'

const EMPTY_FORM = { name: '', buyMonths: '', giftMonths: '' }

export default function GiftRulesTab() {
  const t = useT()
  const m = t.pages.marketing
  const confirmDialog = useConfirm()
  const [items, setItems] = useState<GiftRule[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [open, setOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState(EMPTY_FORM)

  const load = () => {
    setError('')
    listGiftRules()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const closeForm = () => {
    setOpen(false)
    setForm(EMPTY_FORM)
    setFormError('')
  }

  const submit = async () => {
    if (creating) return
    setFormError('')
    const buy = Number(form.buyMonths), gift = Number(form.giftMonths)
    if (!form.name.trim() || !buy || !gift || buy < 1 || gift < 1) {
      setFormError(m.formIncomplete); return
    }
    setCreating(true)
    try {
      await createGiftRule({ legalEntityId: 1, name: form.name.trim(), buyMonths: buy, giftMonths: gift })
      toast.success(m.createdOk)
      closeForm()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setCreating(false)
    }
  }

  const disable = async (id: number, name: string) => {
    if (!(await confirmDialog(m.disableConfirm.replace('{name}', name), { danger: true }))) return
    try { await disableGiftRule(id); load() } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  const columns: ColumnDef[] = [
    { key: 'name', label: m.colName, render: (r) => <span className="font-medium">{String(r.name ?? '—')}</span> },
    { key: 'buyMonths', label: m.giftBuyMonths, render: (r) => String(r.buyMonths ?? '—') },
    { key: 'giftMonths', label: m.giftGiftMonths, render: (r) => String(r.giftMonths ?? '—') },
    { key: 'status', label: m.colStatus, render: (r) => (
      <RuleStatus status={String(r.status)} />
    ) },
    { key: 'op', label: m.colOp, render: (r) => (r.status === 'ENABLED'
      ? <ActionLink onClick={() => disable(Number(r.ruleId), String(r.name))} label={m.disable} />
      : null) },
  ]

  const paged = items.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div>
      <PageHead title={m.tabGift} desc={m.desc} />
      <Card className="p-4">
        <div className="mb-3 flex justify-end">
          <ToolbarButton primary onClick={() => setOpen(true)}>+ {m.create}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <DataTable columns={columns} rows={paged.map((r) => ({ ...r }))} emptyText={m.empty} />
        )}
        {!error && items.length > 0 && (
          <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={items.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(m)} />
          </div>
        )}
      </Card>
      {open && (
        <Drawer title={m.create} onClose={closeForm}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={closeForm}>
                {t.common.confirmDialog.cancel}
              </button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={creating} onClick={submit}>
                {creating ? m.creating : m.create}
              </button>
            </>
          }>
          <div className="grid gap-3">
            <FormField label={m.colName} required>
              <Input value={form.name} placeholder={m.giftNamePh}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
            <FormField label={m.giftBuyMonths} required>
              <Input inputMode="numeric" value={form.buyMonths} placeholder={m.giftBuyPh}
                onChange={(e) => setForm({ ...form, buyMonths: e.target.value })} />
            </FormField>
            <FormField label={m.giftGiftMonths} required>
              <Input inputMode="numeric" value={form.giftMonths} placeholder={m.giftGiftPh}
                onChange={(e) => setForm({ ...form, giftMonths: e.target.value })} />
            </FormField>
          </div>
          {formError && <div className="mt-3"><ErrorBanner message={formError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
