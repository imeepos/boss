// 赠送时长阶梯规则 Tab:列表 + 抽屉式新建 + 停用(6送1/12送3/24送6 类规则)。
import { useEffect, useState } from 'react'
import { listGiftRules, createGiftRule, disableGiftRule, type GiftRule } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton, FormField } from '../../../components/business'
import { Drawer } from '../../../components/Drawer'

const EMPTY_FORM = { name: '', buyMonths: '', giftMonths: '' }

export default function GiftRulesTab() {
  const t = useT()
  const m = t.pages.marketing
  const [items, setItems] = useState<GiftRule[]>([])
  const [error, setError] = useState('')
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
      closeForm()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setCreating(false)
    }
  }

  const disable = async (id: number) => {
    try { await disableGiftRule(id); load() } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  return (
    <div>
      <Card className="p-4">
        <div className="mb-3 flex justify-end">
          <ToolbarButton primary onClick={() => setOpen(true)}>+ {m.create}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{m.colName}</TableHead>
                <TableHead>{m.giftBuyMonths}</TableHead>
                <TableHead>{m.giftGiftMonths}</TableHead>
                <TableHead>{m.colStatus}</TableHead>
                <TableHead>{m.colOp}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((r) => (
                <TableRow key={r.ruleId}>
                  <TableCell className="font-medium">{r.name}</TableCell>
                  <TableCell>{r.buyMonths}</TableCell>
                  <TableCell>{r.giftMonths}</TableCell>
                  <TableCell>
                    <Badge variant={r.status === 'ENABLED' ? 'success' : 'default'}>{r.status}</Badge>
                  </TableCell>
                  <TableCell>
                    {r.status === 'ENABLED' && (
                      <button className="text-xs text-[var(--color-text-link)] hover:underline"
                        onClick={() => disable(r.ruleId)}>{m.disable}</button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={5}><EmptyState text={m.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
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
              <input className="w-full" value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
            <FormField label={m.giftBuyMonths} required>
              <input className="w-full" inputMode="numeric" value={form.buyMonths}
                onChange={(e) => setForm({ ...form, buyMonths: e.target.value })} />
            </FormField>
            <FormField label={m.giftGiftMonths} required>
              <input className="w-full" inputMode="numeric" value={form.giftMonths}
                onChange={(e) => setForm({ ...form, giftMonths: e.target.value })} />
            </FormField>
          </div>
          {formError && <div className="mt-3"><ErrorBanner message={formError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
