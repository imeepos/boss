// 券模板管理 Tab:列表 + 抽屉式新建(placeholder/类型 tip/toast/按钮微反馈) + 停用。单位见 promotion.yaml。
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import {
  listCouponTemplates, createCouponTemplate, disableCouponTemplate, type CouponTemplate,
} from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { RuleStatus } from './RuleStatus'
import {
  PageHead, pagerTexts, ErrorBanner, ToolbarButton, FormField, SubmitButton,
  ActionLink, DataTable, type ColumnDef, type SubmitState,
} from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
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
  const confirmDialog = useConfirm()
  const typeOpts = typeOptions(m)
  const [items, setItems] = useState<CouponTemplate[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
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

  const disable = async (id: number, name: string) => {
    if (!(await confirmDialog(m.disableConfirm.replace('{name}', name), { danger: true }))) return
    try { await disableCouponTemplate(id); toast.success(m.disabledOk); load() } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  const columns: ColumnDef[] = [
    { key: 'name', label: m.colName, render: (r) => <span className="font-medium">{String(r.name ?? '—')}</span> },
    { key: 'type', label: m.couponType, render: (r) => typeOpts.find((o) => o.value === r.type)?.label ?? String(r.type) },
    { key: 'faceValue', label: m.couponFaceYuan, render: (r) => yuan(Number(r.faceValue)) },
    { key: 'threshold', label: m.couponThresholdYuan, render: (r) => (Number(r.threshold) > 0 ? yuan(Number(r.threshold)) : '-') },
    { key: 'validDays', label: m.couponValidDays, render: (r) => (Number(r.validDays) > 0 ? String(r.validDays) : '-') },
    { key: 'issued', label: m.couponIssued, render: (r) => (Number(r.totalQty) > 0 ? Number(r.issuedQty) + '/' + Number(r.totalQty) : String(r.issuedQty)) },
    { key: 'status', label: m.colStatus, render: (r) => (
      <RuleStatus status={String(r.status)} />
    ) },
    { key: 'op', label: m.colOp, render: (r) => (r.status === 'ENABLED'
      ? <ActionLink onClick={() => disable(Number(r.templateId), String(r.name))} label={m.disable} />
      : null) },
  ]

  const paged = items.slice((page - 1) * pageSize, page * pageSize)
  const typeTip = { CASH: m.couponTypeTipCash, FULL_CUT: m.couponTypeTipFullCut, DISCOUNT: m.couponTypeTipDiscount }[form.type]

  return (
    <div>
      <PageHead title={m.tabCoupons} desc={m.desc} />
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
