// 供应商管理抽屉:列表 + 新增(POST)+ 编辑(PUT /procurement/suppliers/{id})
// + 停用(POST /:id/disable)+ 启用(POST /:id/enable),均 ConfirmDialog/校验走 purchaseLogic。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Drawer } from '../../../components/Drawer'
import { Button } from '../../../components/ui/button'
import type { SupplierRow } from '../types'
import { buildSupplierPayload, canEnableSupplier, emptySupplierForm, supplierFormErr, type SupplierFormState } from './purchaseLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const smallBtn = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50'
const td = 'h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'

export function SuppliersDrawer({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const t = useT()
  const d = t.pages.purchasePage
  const confirm = useConfirm()
  const [rows, setRows] = useState<SupplierRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [editing, setEditing] = useState<SupplierRow | null>(null)
  const [form, setForm] = useState<SupplierFormState>(emptySupplierForm)

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: SupplierRow[] }>('/procurement/suppliers')
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(load, [load])

  const startEdit = (r: SupplierRow) => {
    setEditing(r)
    setForm({
      code: r.code, name: r.name, contactName: r.contactName,
      contactPhone: r.contactPhone, legalEntityId: r.legalEntityId, remark: r.remark ?? '',
    })
  }
  const resetForm = () => { setEditing(null); setForm(emptySupplierForm) }

  const submit = async () => {
    if (busy) return
    if (supplierFormErr(form)) { toast.error(d.supErrName); return }
    setBusy(true)
    try {
      await apiFetch(editing ? '/procurement/suppliers/' + editing.id : '/procurement/suppliers', {
        method: editing ? 'PUT' : 'POST',
        body: buildSupplierPayload(form),
      })
      toast.success(d.supSaveOk)
      resetForm()
      load()
      onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : d.opFail)
    } finally {
      setBusy(false)
    }
  }

  const disable = async (r: SupplierRow) => {
    const ok = await confirm(d.supDisableConfirm.replace('{name}', r.name), { danger: true, title: d.supDisable })
    if (!ok) return
    try {
      await apiFetch('/procurement/suppliers/' + r.id + '/disable', { method: 'POST' })
      toast.success(d.supDisable)
      load()
      onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : d.opFail)
    }
  }

  const enable = async (r: SupplierRow) => {
    const ok = await confirm(d.supEnableConfirm.replace('{name}', r.name), { title: d.supEnable })
    if (!ok) return
    setBusy(true)
    try {
      await apiFetch('/procurement/suppliers/' + r.id + '/enable', { method: 'POST' })
      toast.success(d.supEnable)
      load()
      onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : d.opFail)
    } finally {
      setBusy(false)
    }
  }

  const cols = [d.supName, d.supCode, d.supContact, d.supPhone, d.colStatus, d.colActions]
  return (
    <Drawer title={d.suppliersTitle} onClose={onClose} width={680}
      footer={<Button variant="outline" size="sm" className="h-8 px-4 text-[13px]" onClick={onClose}>{d.cancel}</Button>}>
      <div className="flex flex-col gap-3">
        <div className="rounded-sm border border-[var(--shell-side-border)] p-3">
          <div className="mb-2 text-[13px] font-medium text-[var(--shell-heading)]">{editing ? d.supEdit : d.supCreate}</div>
          <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
            <input className={input} placeholder={d.supName} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            <input className={input} placeholder={d.supCode} value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} />
            <input className={input} placeholder={d.supContact} value={form.contactName} onChange={(e) => setForm({ ...form, contactName: e.target.value })} />
            <input className={input} placeholder={d.supPhone} value={form.contactPhone} onChange={(e) => setForm({ ...form, contactPhone: e.target.value })} />
            <input className={input} type="number" placeholder={d.supEntity} value={form.legalEntityId} onChange={(e) => setForm({ ...form, legalEntityId: Number(e.target.value) })} />
            <input className={input} placeholder={d.remark} value={form.remark} onChange={(e) => setForm({ ...form, remark: e.target.value })} />
          </div>
          <div className="mt-2 flex items-center gap-2">
            <Button size="sm" className="h-8 px-4 text-[13px]" disabled={busy} onClick={submit}>{editing ? d.save : d.supCreate}</Button>
            {editing && <button className={smallBtn} onClick={resetForm}>{d.cancel}</button>}
          </div>
        </div>
        {error ? (
          <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr>{cols.map((x) => <th key={x} className="h-9 px-2 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className={td}>{r.name}</td>
                    <td className={td + ' font-mono'}>{r.code}</td>
                    <td className={td}>{r.contactName || '—'}</td>
                    <td className={td}>{r.contactPhone || '—'}</td>
                    <td className={td}>
                      <span className={r.status === 'ENABLED' ? 'text-[var(--color-success)]' : 'text-[var(--shell-group-title)]'}>
                        {r.status === 'ENABLED' ? d.supEnabled : d.supDisabled}
                      </span>
                    </td>
                    <td className={td}>
                      <span className="inline-flex items-center gap-2">
                        <button className={smallBtn} disabled={busy} onClick={() => startEdit(r)}>{d.supEdit}</button>
                        {r.status === 'ENABLED' && (
                          <button className={smallBtn} disabled={busy} onClick={() => disable(r)}>{d.supDisable}</button>
                        )}
                        {canEnableSupplier(r.status) && (
                          <button className={smallBtn} disabled={busy} onClick={() => enable(r)}>{d.supEnable}</button>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!rows.length && <tr><td className={td + ' text-center'} colSpan={6}>{t.common.loading}</td></tr>}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Drawer>
  )
}
