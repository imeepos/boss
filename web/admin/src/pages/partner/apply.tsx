// 企业入驻申请页(公开,免登录):分步表单(企业信息→联系方式→合作意向)。
// 每步本地校验(必填/格式),全部通过才允许提交;双主题/三语言。
// 契约:POST /partner/applications(公开端点);提交成功展示回执号。
import { useState, type CSSProperties, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { submitPartnerApplication } from '../../api/partner'
import { useT } from '../../i18n'
import { AuthShell, BrandAside, inputStyle, buttonStyle } from '../auth-shell'

type FormKey = 'companyName' | 'creditCode' | 'contactName' | 'contactPhone' | 'email' | 'businessDesc'

const STEP_FIELDS: FormKey[][] = [
  ['companyName', 'creditCode'],
  ['contactName', 'contactPhone', 'email'],
  ['businessDesc'],
]

const creditCodeRe = /^[0-9A-Z]{18}$/i
const phoneRe = /^(1\d{10}|0\d{2,3}-?\d{7,8})$/
const emailRe = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const labelStyle: CSSProperties = {
  display: 'block', marginBottom: 4, fontSize: 12, color: 'var(--shell-content-text)',
}
const errStyle: CSSProperties = {
  margin: '4px 0 0', fontSize: 12, color: 'var(--color-danger)',
}
const secondaryBtn: CSSProperties = {
  ...buttonStyle, letterSpacing: 0,
  background: 'transparent', color: 'var(--shell-content-text)',
  border: '1px solid var(--shell-content-text)',
}

export default function PartnerApplyPage() {
  const t = useT()
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [form, setForm] = useState<Record<FormKey, string>>({
    companyName: '', creditCode: '', contactName: '', contactPhone: '', email: '', businessDesc: '',
  })
  const [errors, setErrors] = useState<Partial<Record<FormKey, string>>>({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [receiptId, setReceiptId] = useState<number | null>(null)

  const set = (k: FormKey, v: string) => {
    setForm({ ...form, [k]: v })
    if (errors[k]) setErrors({ ...errors, [k]: undefined })
  }

  const validateField = (k: FormKey): string | undefined => {
    const v = form[k].trim()
    if (k === 'email') {
      if (v === '') return undefined
      return emailRe.test(v) ? undefined : t.pages.partnerApply.invalidEmail
    }
    if (v === '') return t.pages.partnerApply.fieldRequired
    if (k === 'creditCode' && !creditCodeRe.test(v)) return t.pages.partnerApply.invalidCreditCode
    if (k === 'contactPhone' && !phoneRe.test(v)) return t.pages.partnerApply.invalidPhone
    return undefined
  }

  const validateStep = (idx: number): boolean => {
    const next: Partial<Record<FormKey, string>> = {}
    for (const k of STEP_FIELDS[idx]) {
      const msg = validateField(k)
      if (msg) next[k] = msg
    }
    setErrors(next)
    return Object.keys(next).length === 0
  }

  const goNext = () => {
    if (validateStep(step)) setStep(Math.min(step + 1, STEP_FIELDS.length - 1))
  }
  const goPrev = () => setStep(Math.max(step - 1, 0))

  const submit = async () => {
    if (submitting) return
    for (let i = 0; i < STEP_FIELDS.length; i++) {
      if (!validateStep(i)) { setStep(i); return }
    }
    setSubmitting(true)
    setError('')
    try {
      const res = await submitPartnerApplication(form)
      if (!res?.applicationId) throw new Error(t.pages.partnerApply.fail)
      setReceiptId(res.applicationId)
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.partnerApply.fail)
    } finally {
      setSubmitting(false)
    }
  }

  const p = t.pages.partnerApply
  return (
    <AuthShell aside={<BrandAside tip={p.subtitle} />}>
      <div style={{ padding: '56px 40px 0' }}>
      <h2 className="m-0 mb-1 text-xl font-bold text-[var(--shell-heading)]">{p.title}</h2>
      <p className="m-0 mb-5 text-xs text-[var(--shell-crumb-text)]">{p.subtitle}</p>
      {receiptId !== null ? <Receipt id={receiptId} onBack={() => navigate('/login')} /> : (
        <div>
          <StepBar step={step} labels={[p.stepCompany, p.stepContact, p.stepIntent]} />
          <p className="m-0 mb-4 text-xs text-[var(--shell-crumb-text)]">
            {p.stepOf.replace('{n}', String(step + 1)).replace('{total}', String(STEP_FIELDS.length))}
          </p>
          {step === 0 && (
            <>
              <Field label={p.companyName} required error={errors.companyName}>
                <input style={inputStyle} placeholder={p.companyNamePh} value={form.companyName}
                  onChange={(e) => set('companyName', e.target.value)} />
              </Field>
              <Field label={p.creditCode} required error={errors.creditCode}>
                <input style={inputStyle} placeholder={p.creditCodePh} maxLength={18} value={form.creditCode}
                  onChange={(e) => set('creditCode', e.target.value)} />
              </Field>
            </>
          )}
          {step === 1 && (
            <>
              <Field label={p.contactName} required error={errors.contactName}>
                <input style={inputStyle} placeholder={p.contactNamePh} value={form.contactName}
                  onChange={(e) => set('contactName', e.target.value)} />
              </Field>
              <Field label={p.contactPhone} required error={errors.contactPhone}>
                <input style={inputStyle} placeholder={p.contactPhonePh} value={form.contactPhone}
                  onChange={(e) => set('contactPhone', e.target.value)} />
              </Field>
              <Field label={p.email} error={errors.email}>
                <input style={inputStyle} placeholder={p.emailPh} value={form.email}
                  onChange={(e) => set('email', e.target.value)} />
              </Field>
            </>
          )}
          {step === 2 && (
            <Field label={p.businessDesc} required error={errors.businessDesc}>
              <textarea style={{ ...inputStyle, height: 72, paddingTop: 8, resize: 'vertical' }}
                placeholder={p.businessDescPlaceholder} value={form.businessDesc}
                onChange={(e) => set('businessDesc', e.target.value)} />
            </Field>
          )}
          {error && <p className="m-0 mb-3 text-xs text-[var(--color-danger)]">{error}</p>}
          <div className="flex items-center gap-3">
            {step > 0 && <button style={secondaryBtn} onClick={goPrev}>{p.prev}</button>}
            {step < STEP_FIELDS.length - 1 ? (
              <button style={buttonStyle} onClick={goNext}>{p.next}</button>
            ) : (
              <button style={{ ...buttonStyle, letterSpacing: 4 }} disabled={submitting} onClick={submit}>
                {submitting ? p.submitting : p.submit}
              </button>
            )}
          </div>
          <p className="m-0 mt-3 text-center text-xs">
            <a className="cursor-pointer text-[var(--color-text-link)] hover:underline"
              onClick={() => navigate('/login')}>
              {p.backLogin}
            </a>
          </p>
        </div>
      )}
      </div>
    </AuthShell>
  )
}

function Field({ label, required, error, children }: {
  label: string; required?: boolean; error?: string; children: ReactNode
}) {
  return (
    <div className="mb-3">
      <label style={labelStyle}>
        {label}{required && <span style={{ color: 'var(--color-danger)' }}> *</span>}
      </label>
      {children}
      {error && <p style={errStyle}>{error}</p>}
    </div>
  )
}

function StepBar({ step, labels }: { step: number; labels: string[] }) {
  return (
    <div className="mb-4 flex items-center gap-2" role="list" aria-label="steps">
      {labels.map((label, i) => {
        const reached = i <= step
        const active = i === step
        return (
          <div key={label} className="flex flex-1 items-center gap-2" role="listitem">
            <span
              data-step={i}
              className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
              style={{
                background: reached ? 'var(--color-brand-blue-700)' : 'transparent',
                color: reached ? '#fff' : 'var(--shell-content-text)',
                border: reached ? 'none' : '1px solid var(--shell-content-text)',
              }}
            >
              {i < step ? '✓' : i + 1}
            </span>
            <span
              className="text-xs"
              style={{ color: active ? 'var(--shell-heading)' : 'var(--shell-crumb-text)',
                fontWeight: active ? 600 : 400 }}
            >
              {label}
            </span>
            {i < labels.length - 1 && (
              <span className="h-px flex-1" style={{ background: 'var(--shell-crumb-text)', opacity: 0.4 }} />
            )}
          </div>
        )
      })}
    </div>
  )
}

function Receipt({ id, onBack }: { id: number; onBack: () => void }) {
  const t = useT()
  const p = t.pages.partnerApply
  return (
    <div>
      <p className="m-0 mb-2 text-sm font-semibold text-[var(--color-success)]">{p.success}</p>
      <p className="m-0 mb-1 text-xs text-[var(--shell-content-text)]">
        {p.receiptNo}: <b>{id}</b>
      </p>
      <p className="m-0 mb-4 text-xs text-[var(--shell-crumb-text)]">{p.successDesc}</p>
      <button style={buttonStyle} onClick={onBack}>{p.backLogin}</button>
    </div>
  )
}
