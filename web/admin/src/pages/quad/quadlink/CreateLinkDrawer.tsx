// 建链抽屉(P1-E 能力入口):POST /quad-links,五对象全选择器(资产/客户/端口/地址/法人)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { searchCustomers } from '../../../api/pickers'
import { Drawer } from '../../../components/Drawer'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { ErrorBanner } from '../../../components/business/page-head'
import { useT } from '../../../i18n'
import type { LegalEntityRow } from '../../org/company/filter'

type Opt = { value: string; label: string }
interface AssetItem { assetId: number; assetCode: string }
interface PortItem { portId: number; portCode: string }
interface AddrHit { node: { id: number; name: string } }

export function CreateLinkDrawer({ onDone, onClose }: { onDone: () => void; onClose: () => void }) {
  const t = useT()
  const q = t.pages.quadLinkPage
  const [asset, setAsset] = useState('')
  const [customer, setCustomer] = useState('')
  const [port, setPort] = useState('')
  const [address, setAddress] = useState('')
  const [entity, setEntity] = useState('')
  const [entities, setEntities] = useState<LegalEntityRow[]>([])
  const [assets, setAssets] = useState<Opt[]>([])
  const [ports, setPorts] = useState<Opt[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<LegalEntityRow[]>('/legal-entities').then((x) => setEntities(Array.isArray(x) ? x : [])).catch(() => setEntities([]))
    apiFetch<{ items: AssetItem[] }>('/assets').then((d) => setAssets((d?.items ?? []).map((a) => ({ value: String(a.assetId), label: a.assetCode })))).catch(() => setAssets([]))
    apiFetch<{ items: PortItem[] }>('/ports').then((d) => setPorts((d?.items ?? []).map((x) => ({ value: String(x.portId), label: x.portCode })))).catch(() => setPorts([]))
  }, [])

  const searchAddresses = async (kw: string) => {
    const hits = await apiFetch<AddrHit[]>('/addresses/search', { query: { q: kw } }) ?? []
    return hits.map((h) => ({ value: String(h.node.id), label: h.node.name }))
  }

  const save = async () => {
    if (!asset || !customer || !port || !address || !entity) return
    setBusy(true); setError('')
    try {
      await apiFetch('/quad-links', { method: 'POST', body: { assetId: Number(asset), customerId: Number(customer), portId: Number(port), addressId: Number(address), legalEntityId: Number(entity) } })
      toast.success(q.createOk)
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : q.loadFail)
      setBusy(false)
    }
  };

  const pick = (label: string, value: string, onChange: (v: string) => void, props: Record<string, unknown>) => (
    <label className="flex flex-col gap-1.5"><span className="text-xs text-[var(--shell-content-text)]">{label}</span>
      <SimplePicker value={value} onChange={onChange} ariaLabel={label} placeholder={label} searchPlaceholder={label} minWidth={260} {...props} />
    </label>
  );

  return (
    <Drawer title={q.createTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !asset || !customer || !port || !address || !entity} onClick={save}>{busy ? t.pages.account.submitting : q.save}</button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {error && <ErrorBanner message={error} />}
        {pick(q.columns[0] ?? 'Asset', asset, setAsset, { options: assets })}
        {pick(q.columns[1] ?? 'Customer', customer, setCustomer, { search: searchCustomers, toOption: (c: { id: number; name: string; phone?: string | null; customerCode?: string | null }) => ({ value: String(c.id), label: `${c.name} · ${c.phone || c.customerCode || ''}` }) })}
        {pick(q.columns[2] ?? 'Port', port, setPort, { options: ports })}
        {pick(q.columns[3] ?? 'Address', address, setAddress, { search: searchAddresses })}
        {pick(q.legalEntityLabel, entity, setEntity, { options: entities.map((x) => ({ value: String(x.id), label: x.name })) })}
      </div>
    </Drawer>
  );
}
