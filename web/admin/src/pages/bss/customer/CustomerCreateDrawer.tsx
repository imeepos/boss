// 代客开户抽屉:POST /customers(直建,REAL_NAME PENDING/SERVICE ACTIVE 默认)。
// 区域/主体来自 GET /customers/onboarding-catalog;地址用 ResourcePicker 远程检索
// (同目录端点带 q,祖先链已由服务端拍平 fullPath,防抖与竞态由选择器兜底)。
// 树上无目标地址时点「新增地址」在弹框内逐级先搜后建(POST /customers/address,
// fields.md §1.5.0c;回执钉选回显,表单零丢失),再以此 addressId 开户。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { FormField } from '../../../components/business/form-field'
import { useT } from '../../../i18n'
import { AddressChainDrawer, type ChainPickResult } from '../../boss/order/AddressChainDrawer'

interface RegionOption { id: number; name: string; path: string; legalEntityName: string }
interface EntityOption { id: number; name: string; code: string; isPlatform: boolean }
interface AddressOption { id: number; name: string; fullPath: string }

export function CustomerCreateDrawer({
  open, onClose, onCreated,
}: { open: boolean; onClose: () => void; onCreated: (id: number) => void }) {
  const t = useT()
  const c = t.pages.customer
  const o = t.pages.orderPage
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [idType, setIdType] = useState(c.idTypes[0])
  const [idNo, setIdNo] = useState('')
  const [regionId, setRegionId] = useState(0)
  const [entityId, setEntityId] = useState(0)
  const [addressId, setAddressId] = useState(0)
  const [regions, setRegions] = useState<RegionOption[]>([])
  const [entities, setEntities] = useState<EntityOption[]>([])
  const [chainOpen, setChainOpen] = useState(false)
  const [chainPinned, setChainPinned] = useState<ChainPickResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    apiFetch<{ regions: RegionOption[]; legalEntities: EntityOption[] }>('/customers/onboarding-catalog')
      .then((d) => { setRegions(d?.regions ?? []); setEntities(d?.legalEntities ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : c.createFail))
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  const searchAddresses = (kw: string): Promise<AddressOption[] | null> =>
    kw === ''
      ? Promise.resolve([])
      : apiFetch<{ addresses: AddressOption[] }>('/customers/onboarding-catalog', { query: { q: kw } })
          .then((d) => d?.addresses ?? [])

  const reset = () => {
    setName(''); setPhone(''); setIdType(c.idTypes[0]); setIdNo('')
    setRegionId(0); setEntityId(0); setAddressId(0); setError(''); setChainPinned(null)
  }

  const submit = async () => {
    if (busy) return
    setBusy(true); setError('')
    try {
      const d = await apiFetch<{ id: number }>('/customers', {
        method: 'POST',
        body: {
          name: name.trim(), phone: phone.trim(), idType,
          idNo: idNo.trim() || undefined, regionId, legalEntityId: entityId, addressId,
        },
      })
      onCreated(d?.id ?? 0)
      reset()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.createFail)
    } finally { setBusy(false) }
  }

  // 钉选回显:内联建址结果(新建地址可能不在检索索引,保证回显不依赖搜索)。
  const pinnedOptions = chainPinned ? [{ value: String(chainPinned.addressId), label: chainPinned.fullPath }] : []

  const ok = name.trim() !== '' && phone.trim() !== '' && regionId > 0 && entityId > 0 && addressId > 0

  if (!open) return null
  return (
    <Drawer title={c.createTitle} onClose={() => { reset(); onClose() }}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={() => { reset(); onClose() }}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50"
            disabled={busy || !ok} onClick={submit}>{busy ? t.pages.account.submitting : t.pages.company.save}</button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <div className="grid grid-cols-2 gap-3">
          <FormField label={c.fName} required>
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={name} placeholder={c.pName} onChange={(e) => setName(e.target.value)} />
          </FormField>
          <FormField label={c.fPhone} required>
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={phone} placeholder={c.pPhone} onChange={(e) => setPhone(e.target.value)} />
          </FormField>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <FormField label={c.fIdType}>
            <Dropdown value={idType} options={c.idTypes.map((v) => ({ value: v, label: v }))}
              onChange={setIdType} ariaLabel={c.fIdType} />
          </FormField>
          <FormField label={c.fIdNo}>
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={idNo} placeholder={c.pIdNo} onChange={(e) => setIdNo(e.target.value)} />
          </FormField>
        </div>
        <FormField label={c.fEntity} required>
          <Dropdown value={entityId ? String(entityId) : ''} ariaLabel={c.fEntity}
            options={[{ value: '', label: c.entityPick }, ...entities.map((e) => ({ value: String(e.id), label: e.isPlatform ? `${e.name}(${c.entityPlatform})` : e.name }))]}
            onChange={(v) => setEntityId(Number(v) || 0)} />
        </FormField>
        <FormField label={c.fRegion} required>
          <Dropdown value={regionId ? String(regionId) : ''} ariaLabel={c.fRegion} searchable
            options={[{ value: '', label: c.regionPick }, ...regions.map((r) => ({ value: String(r.id), label: `${r.name}(${r.legalEntityName || r.path})` }))]}
            onChange={(v) => setRegionId(Number(v) || 0)} />
        </FormField>
        <FormField label={c.fAddress} required hint={c.addrHint}>
          <div className="flex items-center gap-2">
            <ResourcePicker<AddressOption>
              value={addressId ? String(addressId) : ''}
              onChange={(v) => setAddressId(Number(v) || 0)}
              search={searchAddresses}
              toOption={(a) => ({ value: String(a.id), label: a.fullPath })}
              ariaLabel={c.fAddress}
              emptyLabel={c.addrPick}
              pinnedOptions={pinnedOptions}
              searchPlaceholder={c.addrSearchPh}
              errorText={c.addrSearchFail}
            />
            <button type="button" aria-label={o.chainEntry} title={o.chainEntry}
              className="h-8 shrink-0 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[12px] text-[var(--shell-content-text)] whitespace-nowrap hover:border-[var(--color-border-focus)]"
              onClick={() => setChainOpen(true)}>{o.chainEntry}</button>
          </div>
        </FormField>
        {error && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      </div>
      {chainOpen && (
        <AddressChainDrawer endpoint="/customers/address"
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
