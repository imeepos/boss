// 券模板管理 Tab:列表 + 新建 + 停用。券类型/门槛/面值单位见 promotion.yaml。
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

const TYPE_OPTIONS = [
  { value: 'CASH', label: '代金券' },
  { value: 'FULL_CUT', label: '满减券' },
  { value: 'DISCOUNT', label: '折扣券' },
]

/** 分转元展示(营销域金额单位一律为分)。 */
function yuan(cents: number): string {
  return (cents / 100).toFixed(2)
}

export default function CouponTemplatesTab() {
  const t = useT()
  const m = t.pages.marketing
  const [items, setItems] = useState<CouponTemplate[]>([])
  const [error, setError] = useState('')
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState({
    name: '', type: 'CASH', faceValueYuan: '', thresholdYuan: '', validDays: '', totalQty: '',
  })

  const load = () => {
    setError('')
    listCouponTemplates()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

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
      setForm({ name: '', type: 'CASH', faceValueYuan: '', thresholdYuan: '', validDays: '', totalQty: '' })
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setCreating(false)
    }
  }

  const disable = async (id: number) => {
    try {
      await disableCouponTemplate(id); load()
    } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  const inputStyle = { width: '100%' }
  return (
    <div>
      <Card className="mb-3 p-4">
        <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
          <FormField label={m.couponName} required>
            <input className="w-full" style={inputStyle} value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </FormField>
          <FormField label={m.couponType}>
            <Dropdown value={form.type} options={TYPE_OPTIONS} ariaLabel={m.couponType}
              onChange={(v) => setForm({ ...form, type: v })} />
          </FormField>
          <FormField label={m.couponFaceYuan} required>
            <input style={inputStyle} inputMode="decimal" value={form.faceValueYuan}
              onChange={(e) => setForm({ ...form, faceValueYuan: e.target.value })} />
          </FormField>
          <FormField label={m.couponThresholdYuan}>
            <input style={inputStyle} inputMode="decimal" value={form.thresholdYuan}
              onChange={(e) => setForm({ ...form, thresholdYuan: e.target.value })} />
          </FormField>
          <FormField label={m.couponValidDays}>
            <input style={inputStyle} inputMode="numeric" value={form.validDays}
              onChange={(e) => setForm({ ...form, validDays: e.target.value })} />
          </FormField>
          <FormField label={m.couponTotalQty}>
            <input style={inputStyle} inputMode="numeric" value={form.totalQty}
              onChange={(e) => setForm({ ...form, totalQty: e.target.value })} />
          </FormField>
        </div>
        {formError && <ErrorBanner message={formError} />}
        <div className="flex justify-end">
          <ToolbarButton onClick={submit}>{creating ? m.creating : m.create}</ToolbarButton>
        </div>
      </Card>
      <Card className="p-4">
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
                  <TableCell>{TYPE_OPTIONS.find((o) => o.value === r.type)?.label ?? r.type}</TableCell>
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
    </div>
  )
}
