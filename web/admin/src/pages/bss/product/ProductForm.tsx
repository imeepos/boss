// 新建/编辑产品抽屉表单:POST /products(create)· PUT /products/{id}(update,customer.yaml)。
// 编辑态只开放基础信息:月费必须走调价台账、状态走上下架、公司归属不可改(防区域包孤儿)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { useT } from '../../../i18n'
import { PRODUCT_CATEGORIES, PRODUCT_STATUSES, type ProductCategory, type ProductStatus } from './types'

export interface ProductFormValues {
  id: number // 0=新建
  legalEntityId: number
  name: string
  bandwidth: string
  category: ProductCategory
  monthlyFee: string
  status: ProductStatus
}

export function emptyProductForm(): ProductFormValues {
  return { id: 0, legalEntityId: 0, name: '', bandwidth: '', category: 'broadband', monthlyFee: '', status: 'DRAFT' }
}

export function ProductFormDrawer({
  open, values, companyName, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: ProductFormValues
  companyName?: string
  onChange: (v: ProductFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const p = t.pages.product
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])
  const editing = values.id > 0

  useEffect(() => {
    if (!open || editing) return // 编辑态公司只读,不拉公司列表
    apiFetch<{ id: number; name: string }[]>('/legal-entities')
      .then((d) => setCompanies(d ?? []))
      .catch(() => setCompanies([]))
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!open) return null
  const nameOk = values.name.trim().length > 0 && values.name.trim().length <= 64
  const feeOk = editing || (values.monthlyFee !== '' && Number(values.monthlyFee) >= 0 && !Number.isNaN(Number(values.monthlyFee)))

  return (
    <Drawer title={editing ? p.editTitle : p.createTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
            disabled={busy || !nameOk || !feeOk || (!editing && !values.legalEntityId)} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <FormField label={p.fCompany} required>
          {editing ? (
            <div className="flex h-8 items-center rounded-sm border border-dashed border-[var(--shell-input-border)] bg-[var(--shell-menu-hover-bg)] px-2.5 text-[13px] text-[var(--shell-group-title)]">{companyName || `#${values.legalEntityId}`}</div>
          ) : (
            <Dropdown
              value={values.legalEntityId ? String(values.legalEntityId) : ''}
              options={[{ value: '', label: p.pCompany }, ...companies.map((le) => ({ value: String(le.id), label: le.name }))]}
              onChange={(v) => onChange({ ...values, legalEntityId: Number(v) || 0 })}
              ariaLabel={p.pCompany}
            />
          )}
        </FormField>
        <FormField label={p.fName} required>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.name} placeholder={p.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="text-[11px] text-[var(--color-danger)]">{p.eName}</span>}
        </FormField>
        <FormField label={p.fCategory}>
          <Dropdown
            value={values.category}
            options={PRODUCT_CATEGORIES.map((c, i) => ({ value: c, label: p.categoryOptions[i] }))}
            onChange={(v) => onChange({ ...values, category: v as ProductCategory })}
            ariaLabel={p.fCategory}
          />
        </FormField>
        <FormField label={p.fBandwidth}>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.bandwidth} placeholder={p.pBandwidth}
            onChange={(e) => onChange({ ...values, bandwidth: e.target.value })} />
        </FormField>
        <FormField label={p.fFee} required>
          {editing ? (
            <div className="flex h-8 items-center rounded-sm border border-dashed border-[var(--shell-input-border)] bg-[var(--shell-menu-hover-bg)] px-2.5 text-[13px] text-[var(--shell-group-title)]">{p.feeLocked}</div>
          ) : (
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.monthlyFee} placeholder="0.00"
              onChange={(e) => onChange({ ...values, monthlyFee: e.target.value })} />
          )}
          {!feeOk && values.monthlyFee !== '' && <span className="text-[11px] text-[var(--color-danger)]">{p.eFee}</span>}
        </FormField>
        {!editing && (
          <FormField label={p.fStatus}>
            <Dropdown
              value={values.status}
              options={PRODUCT_STATUSES.map((s, i) => ({ value: s, label: p.statusOptions[i] }))}
              onChange={(v) => onChange({ ...values, status: v as ProductStatus })}
              ariaLabel={p.fStatus}
            />
          </FormField>
        )}
        {submitError && <ErrorBanner message={submitError} />}
      </div>
    </Drawer>
  )
}
