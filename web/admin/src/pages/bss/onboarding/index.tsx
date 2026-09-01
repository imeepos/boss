// 开户工作台:一个页面完成 代客开户→实名→下单→环节推进→派单 全流程。
// 聚合既有能力:建档/注册审核/实名代录(bss/customer)、代客下单(boss/order)、
// 指派(boss/dispatch);订单按 customerId 过滤(GET /orders?customerId=)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useQueryState } from '../../../lib/useQueryState'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner } from '../../../components/business/page-head'
import { StatusTag } from '../../../components/StatusTag'
import { CustomerPicker } from '../../../components/pickers/CustomerPicker'
import { fmtTime } from '../../../lib/format'
import type { CustomerRow } from '../customer/types'
import type { OrderListRow } from '../../boss/types'
import { CustomerCreateDrawer } from '../customer/CustomerCreateDrawer'
import { RegistrationQueueDrawer } from '../customer/RegistrationQueueDrawer'
import { RealNameCard } from './RealNameCard'
import { OrderAdvanceCard } from './OrderAdvanceCard'
import { DispatchCard } from './DispatchCard'

const CARD = 'mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'

export default function OnboardingPage() {
  const t = useT()
  const w = t.pages.onboardingPage
  // 客户档案行"开户"动作经 /bss/onboarding?customerId=N 直达预选(先建档后开户动线)。
  const [customerId, setCustomerId] = useQueryState('customerId', '')
  const [customer, setCustomer] = useState<CustomerRow | null>(null)
  const [orders, setOrders] = useState<OrderListRow[]>([])
  const [createOpen, setCreateOpen] = useState(false)
  const [regOpen, setRegOpen] = useState(false)
  const [error, setError] = useState('')

  const loadCustomer = (id: string) => {
    if (!id) { setCustomer(null); setOrders([]); return }
    apiFetch<CustomerRow>(`/customers/${id}`)
      .then((d) => setCustomer(d))
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
  }
  const loadOrders = (id: string) => {
    if (!id) { setOrders([]); return }
    apiFetch<{ items: OrderListRow[] }>('/orders', { query: { customerId: id } })
      .then((d) => setOrders(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
  }
  useEffect(() => { loadCustomer(customerId) }, [customerId]) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { loadOrders(customerId) }, [customerId]) // eslint-disable-line react-hooks/exhaustive-deps

  const refreshOrders = () => loadOrders(customerId)
  const refreshCustomer = () => loadCustomer(customerId)

  const realNameDone = customer?.realNameStatus === 'VERIFIED'
  const ordered = orders.length > 0
  const advanced = orders.some((o) => o.stage >= 8 && o.status !== 'CANCELLED')
  const stepOn = [customerId !== '', realNameDone, ordered, advanced]
  const stepHint = [
    customer ? customer.name : w.stepPick,
    realNameDone ? w.verified : w.notVerified,
    ordered ? `${orders.length}` : w.stepTodo,
    advanced ? w.stepAdvanced : w.stepTodo,
  ]

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <div className={CARD}>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <div className="min-w-[260px] flex-1">
            <CustomerPicker value={customerId} onChange={(v) => { setError(''); setCustomerId(v) }} />
          </div>
          <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreateOpen(true)}>{w.createBtn}</button>
          <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setRegOpen(true)}>{w.regBtn}</button>
        </div>
        <div className="flex flex-wrap gap-2 px-4 pb-4">
          {w.steps.map((label, i) => (
            <span key={label} className={'inline-flex h-7 items-center gap-1.5 rounded-full px-3 text-xs ' + (stepOn[i]
              ? 'bg-[color-mix(in_srgb,var(--color-success)_15%,transparent)] text-[var(--color-success)]'
              : 'bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]')}>
              <span className={'inline-block h-1.5 w-1.5 rounded-full ' + (stepOn[i] ? 'bg-[var(--color-success)]' : 'bg-[var(--shell-side-border)]')} />
              {i + 1}. {label} · {stepHint[i]}
            </span>
          ))}
        </div>
        {error && <ErrorBanner message={error} className="mx-4 mb-3" />}
        {customer && (
          <div className="grid grid-cols-2 gap-x-6 gap-y-1 px-4 pb-4 text-[13px] max-[959px]:grid-cols-1 md:grid-cols-4">
            <ProfileItem k={w.profileName} v={customer.name} />
            <ProfileItem k={w.profilePhone} v={customer.phone} />
            <ProfileItem k={w.profileRealName} v={<StatusTag domain="realName" value={customer.realNameStatus} />} />
            <ProfileItem k={w.profileService} v={<StatusTag domain="service" value={customer.serviceStatus} />} />
            <ProfileItem k={w.profileCreatedAt} v={fmtTime(customer.createdAt)} />
          </div>
        )}
      </div>

      <RealNameCard customer={customer} onChanged={refreshCustomer} />
      <OrderAdvanceCard customerId={customerId} orders={orders} onChanged={refreshOrders} />
      <DispatchCard orders={orders} onChanged={refreshOrders} />

      {createOpen && (
        <CustomerCreateDrawer open onClose={() => setCreateOpen(false)} onCreated={(id) => { if (id > 0) setCustomerId(String(id)) }} />
      )}
      {regOpen && (
        <RegistrationQueueDrawer open onClose={() => setRegOpen(false)} onChanged={() => { loadCustomer(customerId); refreshOrders() }} />
      )}
    </div>
  )
}

function ProfileItem({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div className="flex items-baseline gap-2">
      <span className="shrink-0 text-xs text-[var(--shell-group-title)]">{k}</span>
      <span className="truncate text-[var(--shell-content-text)]">{v || '—'}</span>
    </div>
  )
}
