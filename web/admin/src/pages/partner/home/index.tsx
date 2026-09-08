// 我的企业(partner_admin/partner_staff):入驻企业档案卡片。
import { useEffect, useState } from 'react'
import { fetchPartnerProfile, type PartnerProfile } from '../../../api/partner'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { LoadingState } from '../../../components/business/feedback'

export default function PartnerHomePage() {
  const t = useT()
  const [profile, setProfile] = useState<PartnerProfile | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError(''); setBusy(true)
    fetchPartnerProfile()
      .then(setProfile)
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.partnerHome.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const rows: Array<[string, string]> = profile ? [
    [t.pages.partnerHome.companyName, profile.companyName],
    [t.pages.partnerHome.creditCode, profile.creditCode],
    [t.pages.partnerHome.contact, `${profile.contactName} / ${profile.contactPhone}`],
    [t.pages.partnerHome.email, profile.email || '-'],
    [t.pages.partnerHome.appliedAt, profile.appliedAt.slice(0, 10)],
    [t.pages.partnerHome.approvedAt, profile.approvedAt.slice(0, 10)],
  ] : []

  return (
    <div>
      <PageHead title={t.pages.partnerHome.title} desc={t.pages.partnerHome.desc} />
      <div className="mb-3 flex items-center">
        <div className="flex-1" />
        <ToolbarButton onClick={load} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
      </div>
      <Card className="p-5">
        {error ? <ErrorBanner message={error} /> : busy ? <LoadingState /> : !profile ? (
          <EmptyState text={t.pages.partnerHome.loadFail} />
        ) : (
          <dl className="m-0 grid grid-cols-[130px_1fr] gap-x-4 gap-y-3 text-[13px]">
            {rows.map(([k, v]) => (
              <div key={k} className="col-span-2 grid grid-cols-subgrid">
                <dt className="text-[var(--shell-crumb-text)]">{k}</dt>
                <dd className="m-0 text-[var(--shell-heading)]">{v}</dd>
              </div>
            ))}
          </dl>
        )}
      </Card>
    </div>
  )
}
