// 券模板管理 Tab:列表 + 抽屉式新建(placeholder/类型 tip/toast/按钮微反馈) + 停用。单位见 promotion.yaml。
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import {
  listCouponTemplates, createCouponTemplate, disableCouponTemplate, type CouponTemplate,
} from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton, FormField, SubmitButton, type SubmitState } from '../../../components/business'
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

/** 元字符串转分;空串视为 0,非法或负数返回 null(导出仅供单测)。 */
export function toCents(v: string): number | null {
  if (v.trim() === '') return 0
  const n = Number(v)
  return Number.isFinite(n) && n >= 0 ? Math.round(n * 100) : null
}

/** 计数字符串转整数;空串视为 0,非法/负数/小数返回 null(导出仅供单测)。 */
export function toCount(v: string): number | null {
  if (v.trim() === '') return 0
  const n = Number(v)
  return Number.isInteger(n) && n >= 0 ? n : null
}

export default function CouponTemplatesTab() {
  const t = useT()
  const m = t.pages.marketing
  const typeOpts = typeOptions(m)
  const [items, setItems] = useState<CouponTemplate[]>([])
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [submitState, setSubmitState] = useState<SubmitState>('idle')
  const [form, setForm] = useState(EMPTY_FORM)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current) }, [])
  /** 微反馈停留:success/failed 短暂可见后复位/收起。 */
  const armTimer = (fn: () => void, ms: number) => {
    if (timer.current) clearTimeout(timer.current)
    timer.current = setTimeout(fn, ms)
  }

  const load = () => {
    setError('')
    listCouponTemplates()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const closeForm = () => {
    if (timer.current) clearTimeout(timer.current)
    setOpen(false)
    setSubmitState('idle')
    setForm(EMPTY_FORM)
  }

  const submit = () => {
    if (submitState !== 'idle') return
    if (!form.name.trim()) { toast.error(m.formIncomplete); return }
    const face = toCents(form.faceValueYuan)
    if (face === null || face <= 0) { toast.error(m.couponFaceInvalid); return }
    const threshold = toCents(form.thresholdYuan)
    const validDays = toCount(form.validDays)
    const totalQty = toCount(form.totalQty)
    if (threshold === null || validDays === null || totalQty === null) { toast.error(m.couponNumInvalid); return }
    setSubmitState('loading')
    createCouponTemplate({
      legalEntityId: 1,
      name: form.name.trim(),
      type: form.type as CouponTemplate['type'],
      faceValue: face, threshold, validDays, totalQty,
    }).then(() => {
      toast.success(m.couponCreated)
      setSubmitState('success')
      load()
      armTimer(closeForm, 800)
    }).catch((e: unknown) => {
      toast.error(e instanceof Error ? e.message : m.couponCreateFailed)
      setSubmitState('failed')
      armTimer(() => setSubmitState('idle'), 1500)
    })
  }

  const disable = async (id: number) => {
    try { await disableCouponTemplate(id); load() } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  const typeTip = { CASH: m.couponTypeTipCash, FULL_CUT: m.couponTypeTipFullCut, DISCOUNT: m.couponTypeTipDiscount }[form.type]

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
                  <TableCell>{r.totalQty > 0 ? r.issuedQty + '/' + r.totalQty : r.issuedQty}</TableCell>
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
              <SubmitButton state={submitState} onClick={submit}
                labels={{ idle: m.create, loading: m.creating, success: m.couponCreated, failed: m.couponCreateFailed }} />
            </>
          }>
          <div className="grid grid-cols-2 gap-3">
            <FormField label={m.couponName} required>
              <Input value={form.name} placeholder={m.couponNamePh}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
            <FormField label={m.couponType} hint={typeTip}>
              <Dropdown value={form.type} options={typeOpts} ariaLabel={m.couponType}
                onChange={(v) => setForm({ ...form, type: v })} />
            </FormField>
            <FormField label={m.couponFaceYuan} required>
              <Input inputMode="decimal" value={form.faceValueYuan} placeholder={m.couponFacePh}
                onChange={(e) => setForm({ ...form, faceValueYuan: e.target.value })} />
            </FormField>
            <FormField label={m.couponThresholdYuan}>
              <Input inputMode="decimal" value={form.thresholdYuan} placeholder={m.couponThresholdPh}
                onChange={(e) => setForm({ ...form, thresholdYuan: e.target.value })} />
            </FormField>
            <FormField label={m.couponValidDays}>
              <Input inputMode="numeric" value={form.validDays} placeholder={m.couponValidDaysPh}
                onChange={(e) => setForm({ ...form, validDays: e.target.value })} />
            </FormField>
            <FormField label={m.couponTotalQty}>
              <Input inputMode="numeric" value={form.totalQty} placeholder={m.couponTotalQtyPh}
                onChange={(e) => setForm({ ...form, totalQty: e.target.value })} />
            </FormField>
          </div>
        </Drawer>
      )}
    </div>
  )
}