// 企业入驻申请页(公开,免登录):复用登录外壳品牌规格,双主题/三语言。
// 契约:POST /partner/applications(公开端点);提交成功展示回执号。
import { useState, type CSSProperties } from 'react'
import { useNavigate } from 'react-router-dom'
import { submitPartnerApplication } from '../../api/partner'
import { useT } from '../../i18n'
import { AuthShell, BrandAside, inputStyle, buttonStyle } from '../auth-shell'

const labelStyle: CSSProperties = {
  display: 'block',
  marginBottom: 4,
  fontSize: 12,
  color: 'var(--shell-content-text)',
}

export default function PartnerApplyPage() {
  const t = useT()
  const navigate = useNavigate()
  const [form, setForm] = useState({
    companyName: '', creditCode: '', contactName: '', contactPhone: '', email: '', businessDesc: '',
  })
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [receiptId, setReceiptId] = useState<number | null>(null)

  const set = (k: keyof typeof form) => (v: string) => setForm({ ...form, [k]: v })

  const submit = async () => {
    if (submitting) return
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

  return (
    <AuthShell aside={<BrandAside tip={t.pages.partnerApply.subtitle} />}>
      <div style={{ padding: '56px 40px 0' }}>
      <h2 className="m-0 mb-1 text-xl font-bold text-[var(--shell-heading)]">{t.pages.partnerApply.title}</h2>
      <p className="m-0 mb-5 text-xs text-[var(--shell-crumb-text)]">{t.pages.partnerApply.subtitle}</p>
      {receiptId !== null ? (
        <div>
          <p className="m-0 mb-2 text-sm font-semibold text-[var(--color-success)]">
            {t.pages.partnerApply.success}
          </p>
          <p className="m-0 mb-1 text-xs text-[var(--shell-content-text)]">
            {t.pages.partnerApply.receiptNo}: <b>{receiptId}</b>
          </p>
          <p className="m-0 mb-4 text-xs text-[var(--shell-crumb-text)]">{t.pages.partnerApply.successDesc}</p>
          <button style={buttonStyle} onClick={() => navigate('/login')}>
            {t.pages.partnerApply.backLogin}
          </button>
        </div>
      ) : (
        <div>
          <label style={labelStyle}>{t.pages.partnerApply.companyName}</label>
          <input style={inputStyle} value={form.companyName}
            onChange={(e) => set('companyName')(e.target.value)} />
          <label style={labelStyle}>{t.pages.partnerApply.creditCode}</label>
          <input style={inputStyle} value={form.creditCode}
            onChange={(e) => set('creditCode')(e.target.value)} />
          <label style={labelStyle}>{t.pages.partnerApply.contactName}</label>
          <input style={inputStyle} value={form.contactName}
            onChange={(e) => set('contactName')(e.target.value)} />
          <label style={labelStyle}>{t.pages.partnerApply.contactPhone}</label>
          <input style={inputStyle} value={form.contactPhone}
            onChange={(e) => set('contactPhone')(e.target.value)} />
          <label style={labelStyle}>{t.pages.partnerApply.email}</label>
          <input style={inputStyle} value={form.email}
            onChange={(e) => set('email')(e.target.value)} />
          <label style={labelStyle}>{t.pages.partnerApply.businessDesc}</label>
          <textarea style={{ ...inputStyle, height: 72, paddingTop: 8, resize: 'vertical' }}
            placeholder={t.pages.partnerApply.businessDescPlaceholder}
            value={form.businessDesc}
            onChange={(e) => set('businessDesc')(e.target.value)} />
          {error && <p className="m-0 mb-3 text-xs text-[var(--color-danger)]">{error}</p>}
          <button
            style={{ ...buttonStyle, letterSpacing: 4 }}
            disabled={submitting}
            onClick={submit}
          >
            {submitting ? t.pages.partnerApply.submitting : t.pages.partnerApply.submit}
          </button>
          <p className="m-0 mt-3 text-center text-xs">
            <a className="cursor-pointer text-[var(--color-text-link)] hover:underline"
              onClick={() => navigate('/login')}>
              {t.pages.partnerApply.backLogin}
            </a>
          </p>
        </div>
      )}
      </div>
    </AuthShell>
  )
}
