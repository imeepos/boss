// 配置模板编辑抽屉:支持 JSON 内容、状态和版本更新。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { Button } from '../../../components/ui/button'
import { useT } from '../../../i18n'
import type { LegalEntityRow } from '../../org/company/filter'
import type { ProvisionTemplateRow } from '../types'

export function TemplateForm({ entities, initial, onDone, onCancel }: { entities: LegalEntityRow[]; initial?: ProvisionTemplateRow; onDone: () => void; onCancel: () => void }) {
  const p = useT().pages.templatePage
  const [legalEntityId, setLegalEntityId] = useState(initial?.legalEntityId ?? entities[0]?.id ?? 0)
  const [code, setCode] = useState(initial?.code ?? '')
  const [name, setName] = useState(initial?.name ?? '')
  const [status, setStatus] = useState(initial?.status ?? 'ENABLED')
  const [content, setContent] = useState(JSON.stringify(initial?.content ?? {}, null, 2))
  const [err, setErr] = useState(''); const [busy, setBusy] = useState(false)
  const save = async () => {
    let parsed: Record<string, unknown>
    try { parsed = JSON.parse(content) } catch { setErr(p.jsonInvalid); return }
    if (!legalEntityId || !code.trim() || !name.trim()) { setErr(p.requiredHint); return }
    setBusy(true)
    try { await apiFetch(initial ? `/provision-templates/${initial.id}` : '/provision-templates', { method: initial ? 'PUT' : 'POST', body: { legalEntityId, code: code.trim(), name: name.trim(), status, content: parsed } }); toast.success(p.saveOk); onDone() } catch (e) { setErr(e instanceof Error ? e.message : p.saveFail); setBusy(false) }
  }
  return <Drawer title={initial ? p.edit : p.create} onClose={onCancel} footer={<><Button variant="outline" size="sm" className="h-8 px-4 text-[13px]" onClick={onCancel}>{p.cancel}</Button><Button size="sm" className="h-8 px-4 text-[13px]" disabled={busy} onClick={save}>{p.save}</Button></>}>
    {err && <ErrorBanner message={err} />}<div className="flex flex-col gap-3.5"><FormField label={p.legalEntity} required><Dropdown value={String(legalEntityId)} options={entities.map((x) => ({ value: String(x.id), label: `${x.name} (${x.code})` }))} onChange={(v) => setLegalEntityId(Number(v) || 0)} ariaLabel={p.legalEntityPlaceholder} searchable /></FormField><FormField label={p.codeLabel} required><input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px]" value={code} placeholder={p.codePlaceholder} onChange={(e) => setCode(e.target.value)} /></FormField><FormField label={p.nameLabel} required><input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px]" value={name} placeholder={p.namePlaceholder} onChange={(e) => setName(e.target.value)} /></FormField><FormField label={p.statusLabel}><Dropdown value={status} options={[{ value: 'ENABLED', label: 'ENABLED' }, { value: 'DISABLED', label: 'DISABLED' }]} onChange={setStatus} ariaLabel={p.statusLabel} /></FormField><FormField label={p.contentLabel}><textarea className="min-h-40 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-2.5 font-mono text-xs" value={content} onChange={(e) => setContent(e.target.value)} /></FormField></div>
  </Drawer>
}
