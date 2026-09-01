// 新建客户抽屉:POST /customers 轻量建档(REAL_NAME PENDING/SERVICE ACTIVE 默认)。
// 地址不在建档必填之列(000176):先建档,后经档案行"地址"动作内联建址回填,
// 再进开户工作台走 下单→支付→开户→施工→通网;区域/主体来自 GET /customers/onboarding-catalog。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { useT } from '../../../i18n'

interface RegionOption { id: number; name: string; path: string; legalEntityName: string }
interface EntityOption { id: number; name: string; code: string; isPlatform: boolean }

export function CustomerCreateDrawer({
  open, onClose, onCreated,
}: { open: boolean; onClose: () => void; onCreated: (id: number) => void }) {
  const t = useT()
  const c = t.pages.customer
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [idType, setIdType] = useState(c.idTypes[0])
  const [idNo, setIdNo] = useState('')
  const [regionId, setRegionId] = useState(0)
  const [entityId, setEntityId] = useState(0)
  const [regions, setRegions] = useState<RegionOption[]>([])
  const [entities, setEntities] = useState<EntityOption[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    apiFetch<{ regions: RegionOption[]; legalEntities: EntityOption[] }>('/customers/onboarding-catalog')
      .then((d) => { setRegions(d?.regions ?? []); setEntities(d?.legalEntities ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : c.createFail))
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  const reset = () => {
    setName(''); setPhone(''); setIdType(c.idTypes[0]); setIdNo('')
    setRegionId(0); setEntityId(0); setError('')
  }

  const submit = async () => {
    if (busy) return
    setBusy(true); setError('')
    try {
      const d = await apiFetch<{ id: number }>('/customers', {
        method: 'POST',
        body: {
          name: name.trim(), phone: phone.trim(), idType,
          idNo: idNo.trim() || undefined, regionId, legalEntityId: entityId,
        },
      })
      onCreated(d?.id ?? 0)
      reset()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.createFail)
    } finally { setBusy(false) }
  }

  const ok = name.trim() !== '' && phone.trim() !== '' && regionId > 0 && entityId > 0

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
        {error && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      </div>
    </Drawer>
  )
}
