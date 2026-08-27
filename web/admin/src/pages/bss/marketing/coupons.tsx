// 券模板管理 Tab:列表 + 抽屉式新建 + 停用。券类型/门槛/面值单位见 promotion.yaml。
import { useEffect, useState } from 'react'
import {
  listCouponTemplates, createCouponTemplate, disableCouponTemplate, type CouponTemplate,
} from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton, FormField } from '../../../components/business'
import { Dropdown } from '../../../components/Dropdown'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'

const TYPE_VALUES = ['CASH', 'FULL_CUT', 'DISCOUNT'] as const

/** 券类型下拉选项:label 走 i18n,随语言切换。 */
function typeOptions(m: ReturnType<typeof useT>['pages']['marketing']) {
  const labels = { CASH: m.couponTypeCash, FULL_CUT: m.couponTypeFullCut, DISCOUNT: m.couponTypeDiscount }
  return TYPE_VALUES.map((value) => ({ value, label: labels[value] }))
}

const EMPTY_FORM = {
  name: '', type: 'CASH', faceValueYuan: '', thresholdYuan: '', validDays: '', totalQty: '',
}

/** 分转元展示(营销域金额单位一律为分)。 */
function yuan(cents: number): string {
  return (cents / 100).toFixed(2)
}

export default function CouponTemplatesTab() {
  const t = useT()
  const m = t.pages.marketing
  const typeOpts = typeOptions(m)
  const [items, setItems] = useState<CouponTemplate[]>([])
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState(EMPTY_FORM)

  const load = () => {
    setError('')
    listCouponTemplates()
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
    if (!form.name.trim() || !form.faceValueYuan) {
      setFormError(m.formIncomplete); return
    }
    setCreating(true)
    try {
      await createCouponTemplate({
        legalEntityId: 1,
        name: form.name.trim(),
        type: form.type as CouponTemplate['type'],
        faceValue: Math.round(Number(form.faceValueYuan) * 100),
        threshold: Math.round(Number(form.thresholdYuan || 0) * 100),
        validDays: Number(form.validDays || 0),
        totalQty: Number(form.totalQty || 0),
      })
      closeForm()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setCreating(false)
    }
  }

  const disable = async (id: number) => {
    try { await disableCouponTemplate(id); load() } catch (e) {
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
                <TableHead>{m.couponType}</TableHead>
                <TableHead>{m.couponFaceYuan}</TableHead>
                <TableHead>{m.couponThresholdYuan}</TableHead>
                <TableHead>{m.couponValidDays}</TableHead>
                <TableHead>{m.couponIssued}</TableHead>
                <TableHead>{m.colStatus}</TableHead>
                <TableHead>{m.colOp}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((r) => (
                <TableRow key={r.templateId}>
                  <TableCell className="font-medium">{r.name}</TableCell>
                  <TableCell>{typeOpts.find((o) => o.value === r.type)?.label ?? r.type}</TableCell>
                  <TableCell>{yuan(r.faceValue)}</TableCell>
                  <TableCell>{r.threshold > 0 ? yuan(r.threshold) : '-'}</TableCell>
                  <TableCell>{r.validDays > 0 ? r.validDays : '-'}</TableCell>
                  <TableCell>{r.totalQty > 0 ? `${r.issuedQty}/${r.totalQty}` : r.issuedQty}</TableCell>
                  <TableCell>
                    <Badge variant={r.status === 'ENABLED' ? 'success' : 'default'}>{r.status}</Badge>
                  </TableCell>
                  <TableCell>
                    {r.status === 'ENABLED' && (
                      <button className="text-xs text-[var(--color-text-link)] hover:underline"
                        onClick={() => disable(r.templateId)}>{m.disable}</button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={8}><EmptyState text={m.empty} /></TableCell></TableRow>
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
          <div className="grid grid-cols-2 gap-3">
            <FormField label={m.couponName} required>
              <Input value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
            <FormField label={m.couponType}>
              <Dropdown value={form.type} options={typeOpts} ariaLabel={m.couponType}
                onChange={(v) => setForm({ ...form, type: v })} />
            </FormField>
            <FormField label={m.couponFaceYuan} required>
              <Input inputMode="decimal" value={form.faceValueYuan}
                onChange={(e) => setForm({ ...form, faceValueYuan: e.target.value })} />
            </FormField>
            <FormField label={m.couponThresholdYuan}>
              <Input inputMode="decimal" value={form.thresholdYuan}
                onChange={(e) => setForm({ ...form, thresholdYuan: e.target.value })} />
            </FormField>
            <FormField label={m.couponValidDays}>
              <Input inputMode="numeric" value={form.validDays}
                onChange={(e) => setForm({ ...form, validDays: e.target.value })} />
            </FormField>
            <FormField label={m.couponTotalQty}>
              <Input inputMode="numeric" value={form.totalQty}
                onChange={(e) => setForm({ ...form, totalQty: e.target.value })} />
            </FormField>
          </div>
          {formError && <div className="mt-3"><ErrorBanner message={formError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
