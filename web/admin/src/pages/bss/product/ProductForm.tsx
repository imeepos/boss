// 新建产品抽屉表单:POST /products(customer.yaml createProduct)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import { PRODUCT_STATUSES, type ProductStatus } from './types'

export interface ProductFormValues {
  legalEntityId: number
  name: string
  bandwidth: string
  monthlyFee: string
  status: ProductStatus
}

export function emptyProductForm(): ProductFormValues {
  return { legalEntityId: 0, name: '', bandwidth: '', monthlyFee: '', status: 'DRAFT' }
}

export function ProductFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: ProductFormValues
  onChange: (v: ProductFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const p = t.pages.product
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])

  useEffect(() => {
    if (!open) return
    apiFetch<{ id: number; name: string }[]>('/legal-entities')
      .then((d) => setCompanies(d ?? []))
      .catch(() => setCompanies([]))
  }, [open])

  if (!open) return null
  const nameOk = values.name.trim().length > 0 && values.name.trim().length <= 64
  const feeOk = values.monthlyFee !== '' && Number(values.monthlyFee) >= 0 && !Number.isNaN(Number(values.monthlyFee))

  return (
    <Drawer title={p.createTitle} onClose={onClose}
      footer={
        <>
          <button className="org-btn" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="org-btn org-btn-primary"
            disabled={busy || !values.legalEntityId || !nameOk || !feeOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="org-form">
        <div className="org-field">
          <label><span className="req">*</span>{p.fCompany}</label>
          <select className="org-select" value={values.legalEntityId ? String(values.legalEntityId) : ''}
            onChange={(e) => onChange({ ...values, legalEntityId: Number(e.target.value) || 0 })}>
            <option value="">{p.pCompany}</option>
            {companies.map((le) => <option key={le.id} value={le.id}>{le.name}</option>)}
          </select>
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{p.fName}</label>
          <input className="org-input" value={values.name} placeholder={p.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="acc-err">{p.eName}</span>}
        </div>
        <div className="org-field">
          <label>{p.fBandwidth}</label>
          <input className="org-input" value={values.bandwidth} placeholder={p.pBandwidth}
            onChange={(e) => onChange({ ...values, bandwidth: e.target.value })} />
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{p.fFee}</label>
          <input className="org-input" value={values.monthlyFee} placeholder="0.00"
            onChange={(e) => onChange({ ...values, monthlyFee: e.target.value })} />
          {!feeOk && values.monthlyFee !== '' && <span className="acc-err">{p.eFee}</span>}
        </div>
        <div className="org-field">
          <label>{p.fStatus}</label>
          <select className="org-select" value={values.status}
            onChange={(e) => onChange({ ...values, status: e.target.value as ProductStatus })}>
            {PRODUCT_STATUSES.map((s, i) => <option key={s} value={s}>{p.statusOptions[i]}</option>)}
          </select>
        </div>
        {submitError && <div className="org-error" style={{ margin: 0 }}>{submitError}</div>}
      </div>
    </Drawer>
  )
}
