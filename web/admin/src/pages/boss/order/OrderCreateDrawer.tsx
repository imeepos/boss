// 代客下单抽屉:POST /orders(customerId/offerId/addressId/channelId 必填;
// billingMode PREPAID 时 buyMonths 1~60,环节4 当场收预缴)。
// 客户用 pickers/CustomerPicker;产品/渠道来自 GET /orders/catalog(在售过滤,服务端已做);
// 地址默认带出客户档案地址,可检索覆盖。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { CustomerPicker } from '../../../components/pickers/CustomerPicker'
import { FormField } from '../../../components/business/form-field'
import { useT } from '../../../i18n'
import { AddressChainDrawer, type ChainPickResult } from './AddressChainDrawer'
import { OrderServabilityBadge } from './OrderServabilityBadge'

interface CatalogProduct { id: number; name: string; monthlyFee: number; bandwidth: string }
interface CatalogChannel { id: number; code: string; name: string; status: string }
interface AddressOption { id: number; name: string; fullPath: string }

export function OrderCreateDrawer({
  open, onClose, onCreated, fixedCustomerId,
}: { open: boolean; onClose: () => void; onCreated: (orderNo: string) => void; fixedCustomerId?: string }) {
  const t = useT()
  const o = t.pages.orderPage
  const [customerId, setCustomerId] = useState('')
  const [offerId, setOfferId] = useState(0)
  const [addressId, setAddressId] = useState(0)
  const [channelId, setChannelId] = useState(0)
  const [billingMode, setBillingMode] = useState('POSTPAID')
  const [buyMonths, setBuyMonths] = useState('1')
  const [products, setProducts] = useState<CatalogProduct[]>([])
  const [channels, setChannels] = useState<CatalogChannel[]>([])
  const [customerAddr, setCustomerAddr] = useState(0)
  const [chainOpen, setChainOpen] = useState(false)
  const [chainPinned, setChainPinned] = useState<ChainPickResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    if (fixedCustomerId) setCustomerId(fixedCustomerId)
  }, [open, fixedCustomerId]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!open) return
    apiFetch<{ products: CatalogProduct[]; channels: CatalogChannel[] }>('/orders/catalog')
      .then((d) => { setProducts(d?.products ?? []); setChannels(d?.channels ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : o.loadCatalogFail))
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  // 选定客户后带出档案地址作为默认装机地址(可检索覆盖)。
  // 未手动选择地址时直接默认档案地址,保证保存可用(102 UI 实测发现仅提示不生效)。
  useEffect(() => {
    const id = Number(customerId)
    if (!open || !id) { setCustomerAddr(0); return }
    apiFetch<{ addressId: number }>(`/customers/${id}`)
      .then((d) => {
        setCustomerAddr(d?.addressId ?? 0)
        setAddressId((prev) => (prev === 0 && d?.addressId ? d.addressId : prev))
      })
      .catch(() => setCustomerAddr(0))
  }, [customerId, open]) // eslint-disable-line react-hooks/exhaustive-deps

  const searchAddresses = (kw: string): Promise<AddressOption[] | null> =>
    kw === ''
      ? Promise.resolve([])
      : apiFetch<{ addresses: AddressOption[] }>('/orders/catalog', { query: { q: kw } })
          .then((d) => d?.addresses ?? [])

  const reset = () => {
    setCustomerId(''); setOfferId(0); setAddressId(0); setChannelId(0)
    setBillingMode('POSTPAID'); setBuyMonths('1'); setError(''); setChainPinned(null)
  }

  // 钉选回显:档案地址 + 内联建址结果(新建地址不在检索索引时仍保证回显),按 value 去重。
  const pinnedOptions = [
    ...(customerAddr > 0 ? [{ value: String(customerAddr), label: o.addrPinned.replace('{id}', String(customerAddr)) }] : []),
    ...(chainPinned ? [{ value: String(chainPinned.addressId), label: chainPinned.fullPath }] : []),
  ].filter((p, i, arr) => arr.findIndex((x) => x.value === p.value) === i)

  const monthsOk = billingMode !== 'PREPAID' || (/^\d+$/.test(buyMonths) && Number(buyMonths) >= 1 && Number(buyMonths) <= 60)
  const ok = Number(customerId) > 0 && offerId > 0 && addressId > 0 && channelId > 0 && monthsOk

  const submit = async () => {
    if (busy) return
    setBusy(true); setError('')
    try {
      const d = await apiFetch<{ orderNo: string }>('/orders', {
        method: 'POST',
        body: {
          customerId: Number(customerId), offerId, addressId, channelId,
          billingMode, buyMonths: billingMode === 'PREPAID' ? Number(buyMonths) : 0,
        },
      })
      onCreated(d?.orderNo ?? '')
      reset()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : o.actionFail)
    } finally { setBusy(false) }
  }

  if (!open) return null
  return (
    <Drawer title={o.createTitle} onClose={() => { reset(); onClose() }}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={() => { reset(); onClose() }}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50"
            disabled={busy || !ok} onClick={submit}>{busy ? t.pages.account.submitting : t.pages.company.save}</button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <FormField label={o.fCustomer} required>
          {fixedCustomerId ? (
            <div className="flex h-8 items-center rounded-sm border border-dashed border-[var(--shell-input-border)] bg-[var(--shell-menu-hover-bg)] px-2.5 text-[13px] text-[var(--shell-group-title)]">#{fixedCustomerId}</div>
          ) : (
            <CustomerPicker value={customerId} onChange={setCustomerId} />
          )}
        </FormField>
        <FormField label={o.fProduct} required hint={o.fProductHint}>
          <Dropdown value={offerId ? String(offerId) : ''} ariaLabel={o.fProduct} searchable
            options={[{ value: '', label: o.productPick }, ...products.map((p) => ({ value: String(p.id), label: `${p.name} · ${p.monthlyFee}/月` }))]}
            onChange={(v) => setOfferId(Number(v) || 0)} />
        </FormField>
        <FormField label={o.fAddress} required hint={customerAddr > 0 ? o.fAddressHint.replace('{id}', String(customerAddr)) : undefined}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <ResourcePicker<AddressOption>
                value={addressId ? String(addressId) : ''}
                onChange={(v) => setAddressId(Number(v) || 0)}
                search={searchAddresses}
                toOption={(a) => ({ value: String(a.id), label: a.fullPath })}
                ariaLabel={o.fAddress}
                emptyLabel={o.addrPick}
                pinnedOptions={pinnedOptions}
                searchPlaceholder={o.addrSearchPh}
                errorText={o.addrSearchFail}
              />
              <button type="button" aria-label={o.chainEntry} title={customerId ? undefined : o.chainNeedCustomer}
                disabled={!customerId}
                className="h-8 shrink-0 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[12px] text-[var(--shell-content-text)] whitespace-nowrap disabled:cursor-not-allowed disabled:opacity-50 hover:border-[var(--color-border-focus)] enabled:cursor-pointer"
                onClick={() => setChainOpen(true)}>{o.chainEntry}</button>
            </div>
            {/* T14-2:地址选定即自动判定可装性,失败不阻断下单 */}
            <OrderServabilityBadge addressId={addressId} />
          </div>
        </FormField>
        <div className="grid grid-cols-2 gap-3">
          <FormField label={o.fChannel} required>
            <Dropdown value={channelId ? String(channelId) : ''} ariaLabel={o.fChannel}
              options={[{ value: '', label: o.channelPick }, ...channels.filter((ch) => ch.status === 'ACTIVE').map((ch) => ({ value: String(ch.id), label: ch.name }))]}
              onChange={(v) => setChannelId(Number(v) || 0)} />
          </FormField>
          <FormField label={o.fBillingMode}>
            <Dropdown value={billingMode} ariaLabel={o.fBillingMode}
              options={o.billingOptions.map((label, i) => ({ value: i === 0 ? 'POSTPAID' : 'PREPAID', label }))}
              onChange={setBillingMode} />
          </FormField>
        </div>
        {billingMode === 'PREPAID' && (
          <FormField label={o.fBuyMonths} required hint={o.fBuyMonthsHint} error={monthsOk ? undefined : o.eBuyMonths}>
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={buyMonths} placeholder="1" onChange={(e) => setBuyMonths(e.target.value)} />
          </FormField>
        )}
        {error && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      </div>
      {chainOpen && (
        <AddressChainDrawer customerId={customerId} customerAddressId={customerAddr}
          onDone={(r) => {
            setChainPinned(r)
            setAddressId(r.addressId)
            setChainOpen(false)
          }}
          onClose={() => setChainOpen(false)} />
      )}
    </Drawer>
  )
}
