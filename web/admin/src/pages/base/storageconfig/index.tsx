import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { FormField } from '../../../components/business/form-field'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Switch } from '../../../components/ui/switch'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader,
  DialogFooter, DialogTitle, DialogDescription, DialogClose,
} from '../../../components/ui/dialog'

type Field = { value: string; hasValue: boolean }
type Fields = Record<string, Field>
type Draft = Record<string, string>

function SecretInput({ value, hasValue, onChange }: { value: string; hasValue: boolean; onChange: (v: string) => void }) {
  const [visible, setVisible] = useState(false)
  return (
    <span className="flex w-80 items-center gap-1">
      <Input className="w-72" type={visible ? 'text' : 'password'} value={value} onChange={(e) => onChange(e.target.value)} placeholder={hasValue ? '已配置，不回显' : ''} />
      <button type="button" className="cursor-pointer border-none bg-none px-1 py-0.5 text-xs text-[var(--shell-group-title)]" onClick={() => setVisible((v) => !v)}>{visible ? '隐藏' : '显示'}</button>
    </span>
  )
}

export default function StorageConfigPage() {
  const t = useT()
  const a = t.pages.storageconfig
  const [fields, setFields] = useState<Fields>({})
  const [draft, setDraft] = useState<Draft>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const load = () => {
    setError('')
    apiFetch<{ fields: Fields }>('/storage-config')
      .then((d) => {
        const next = d?.fields ?? {}
        setFields(next)
        setDraft(Object.fromEntries(Object.entries(next).map(([key, field]) => [key, field.value])))
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const set = (key: string, value: string) => setDraft((current) => ({ ...current, [key]: value }))
  const useSSL = draft['minio.useSSL'] === 'true'
  const save = async () => {
    if (saving) return
    setSaving(true)
    try {
      await apiFetch('/storage-config', { method: 'PUT', body: { values: draft } })
      toast.success(a.saved)
      load()
    } catch (e) { toast.error(e instanceof Error ? e.message : a.saveFail) }
    finally { setSaving(false) }
  }

  // 轮换密码
  const [rotateOpen, setRotateOpen] = useState(false)
  const [newSecret, setNewSecret] = useState('')
  const [rotating, setRotating] = useState(false)
  const doRotate = async () => {
    if (rotating || !newSecret) return
    setRotating(true)
    try {
      await apiFetch('/storage-config/rotate-secret', { method: 'POST', body: { secret: newSecret } })
      toast.success(a.rotated)
      setRotateOpen(false)
      setNewSecret('')
      load()
    } catch (e) { toast.error(e instanceof Error ? e.message : a.rotateFail) }
    finally { setRotating(false) }
  }

  if (error) return <div><PageHead title={a.title} desc={a.desc} /><ErrorBanner message={error} /><div className="mt-3"><ToolbarButton onClick={load}>{a.retry}</ToolbarButton></div></div>

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <Card className="p-4">
        <div className="mb-3 flex items-center justify-between">
          <div>
            <div className="font-semibold text-[var(--shell-heading)]">{a.cardTitle}</div>
            <div className="mt-1 text-xs text-[var(--shell-crumb-text)]">{a.cardDesc}</div>
          </div>
          <Badge variant={fields['minio.endpoint']?.hasValue ? 'success' : 'warning'}>{fields['minio.endpoint']?.hasValue ? a.configured : a.notConfigured}</Badge>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <FormField label={a.endpoint} hint={a.endpointHint}><Input className="w-72" value={draft['minio.endpoint'] ?? ''} onChange={(e) => set('minio.endpoint', e.target.value)} placeholder="192.168.0.102:29000" /></FormField>
          <FormField label={a.bucket}><Input className="w-72" value={draft['minio.bucket'] ?? ''} onChange={(e) => set('minio.bucket', e.target.value)} placeholder="boss-attachments" /></FormField>
          <FormField label={a.accessKey}><Input className="w-72" value={draft['minio.accessKey'] ?? ''} onChange={(e) => set('minio.accessKey', e.target.value)} /></FormField>
          <FormField label={a.secretKey}><SecretInput value={draft['minio.secretKey'] ?? ''} hasValue={!!fields['minio.secretKey']?.hasValue} onChange={(v) => set('minio.secretKey', v)} /></FormField>
          <FormField label={a.useSSL}><span className="flex items-center gap-2"><Switch checked={useSSL} onCheckedChange={(v) => set('minio.useSSL', String(v))} aria-label={a.useSSL} /><span className="text-sm">{useSSL ? a.enabled : a.disabled}</span></span></FormField>
        </div>
        <div className="mt-5 flex items-center justify-between">
          <Dialog open={rotateOpen} onOpenChange={setRotateOpen}>
            <DialogTrigger asChild>
              <ToolbarButton>{a.rotateSecret}</ToolbarButton>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{a.rotateSecret}</DialogTitle>
                <DialogDescription>{a.rotateSecretDesc}</DialogDescription>
              </DialogHeader>
              <FormField label={a.newSecret}>
                <Input
                  type="password"
                  value={newSecret}
                  onChange={(e) => setNewSecret(e.target.value)}
                  placeholder={a.newSecretPlaceholder}
                  className="w-full"
                  autoFocus
                />
              </FormField>
              <DialogFooter>
                <DialogClose asChild><ToolbarButton>{a.cancel}</ToolbarButton></DialogClose>
                <ToolbarButton primary disabled={rotating || newSecret.length < 8} onClick={doRotate}>
                  {rotating ? a.rotating : a.rotate}
                </ToolbarButton>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <ToolbarButton primary disabled={saving} onClick={save}>{saving ? a.saving : a.save}</ToolbarButton>
        </div>
      </Card>
    </div>
  )
}
