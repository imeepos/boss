// 系统授权页:展示授权状态(产品/证书有效期/绑定设备),粘贴激活码完成激活。
// 这是未授权时唯一可用的业务页(license/status 与 activate 豁免门禁)。
import { useEffect, useState } from 'react'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { ToolbarButton, CopyButton } from '../../../components/business'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { fmtTime } from '../../../lib/format'
import { fetchLicenseStatus, activateLicense, type LicenseStatus } from '../../../api/license'

export default function LicensePage() {
  const t = useT()
  const s = t.pages.licensePage
  const [status, setStatus] = useState<LicenseStatus | null>(null)
  const [busy, setBusy] = useState(false)
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [activating, setActivating] = useState(false)

  const load = () => {
    setBusy(true); setError('')
    fetchLicenseStatus()
      .then((st) => setStatus(st))
      .catch(() => setError(s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])

  const submit = async () => {
    if (!code.trim()) { setError(s.codeRequired); return }
    setActivating(true); setError('')
    try {
      const st = await activateLicense(code.trim())
      setStatus(st)
      setCode('')
    } catch (e) {
      setError(e instanceof Error ? e.message : s.activateFail)
    } finally {
      setActivating(false)
    }
  }

  if (status?.enabled === false) {
    return (
      <div>
        <PageHead title={s.title} desc={s.desc} />
        <p className="mt-6 text-sm text-[var(--shell-crumb-text)]">{s.notEnabled}</p>
      </div>
    )
  }

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <Card className="mt-6 max-w-3xl p-6">
        <div className="flex items-center gap-3">
          <span className={`inline-flex h-2.5 w-2.5 rounded-full ${status?.activated ? 'bg-[var(--color-success)]' : 'bg-[var(--color-warning)]'}`} />
          <span className="text-sm font-medium text-[var(--shell-input-text)]">
            {status?.activated ? s.activeState : s.inactiveState}
          </span>
        </div>
        <dl className="mt-4 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
          <Row label={s.fLicenseId} value={status?.licenseId ?? '-'} />
          <Row label={s.fProductId} value={status?.productId ?? '-'} />
          <Row label={s.fLicenseType} value={status?.licenseType ?? '-'} />
          <Row label={s.fDeviceId} value={status?.deviceId ?? '-'} />
          <Row label={s.fExpiresAt} value={status?.expiresAt ? fmtTime(status.expiresAt) : '-'} />
          <Row label={s.fGrace} value={status?.inGrace ? s.inGraceYes : s.inGraceNo} />
        </dl>
        {status?.activated && status?.reason && (
          <p className="mt-4 text-sm text-[var(--color-warning)]">{status.reason}</p>
        )}
      </Card>
      <Card className="mt-6 max-w-3xl p-6">
        <h3 className="text-sm font-medium text-[var(--shell-input-text)]">{s.activateTitle}</h3>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center">
          <Input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={s.codePlaceholder}
            className="flex-1"
          />
          <ToolbarButton primary onClick={submit} disabled={activating || busy}>
            {activating ? s.activating : s.activateBtn}
          </ToolbarButton>
        </div>
        {error && (
          <div className="mt-3 flex items-center justify-between gap-3">
            <p className="text-sm break-all text-[var(--color-danger)]">{error}</p>
            <CopyButton text={error} className="h-6 shrink-0 border-none bg-none px-1 text-[11px]" />
          </div>
        )}
      </Card>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4 border-b border-[var(--shell-card-border)] py-2">
      <dt className="text-[var(--shell-crumb-text)]">{label}</dt>
      <dd className="text-right text-[var(--shell-input-text)]">{value}</dd>
    </div>
  )
}