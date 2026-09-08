// 实名卡:展示实名状态;代录入口复用 RealNameDrawer(代录提交与后台核验
// 通过/驳回统一在抽屉内完成,latest PENDING 时抽屉显示核验动作)。
import { useState } from 'react'
import { StatusTag } from '../../../components/StatusTag'
import { Card } from '../../../components/ui/card'
import { RealNameDrawer } from '../customer/RealNameDrawer'
import type { CustomerRow } from '../customer/types'
import { useT } from '../../../i18n'

export function RealNameCard({
  customer, onChanged,
}: { customer: CustomerRow | null; onChanged: () => void }) {
  const t = useT()
  const w = t.pages.onboardingPage
  const [rnOpen, setRnOpen] = useState(false)

  if (!customer) return null

  return (
    <Card>
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="text-[13px] font-medium text-[var(--shell-heading)]">2. {w.realNameTitle}</span>
        <StatusTag domain="realName" value={customer.realNameStatus} />
        <span className="spacer" />
        <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setRnOpen(true)}>{w.realNameEntry}</button>
      </div>
      {rnOpen && (
        <RealNameDrawer customerId={customer.id} customerName={customer.name} onClose={() => setRnOpen(false)} onSubmitted={onChanged} />
      )}
    </Card>
  )
}
