// 负责区域配置对话框(000175):一师傅多负责区域,覆盖式保存;
// 首个选中为主区域(镜像 workers.region_id),复用 Shell/compact 视觉。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { MultiSelect } from '../../../components/MultiSelect'
import { Button } from '../../../components/ui/button'
import type { WorkerRow } from '../types'
import { Err, Shell, compact } from './TeamDialogs'
import { loadRegionOptions, type RegionOption } from './WorkerDialogs'

interface RegionsDialogProps {
  worker: WorkerRow
  onClose: () => void
  onDone: () => void
}

export function WorkerRegionsDialog({ worker, onClose, onDone }: RegionsDialogProps) {
  const w = useT().pages.workerPage
  const [regionIds, setRegionIds] = useState<string[]>((worker.regionIds ?? []).map(String))
  const [regions, setRegions] = useState<RegionOption[]>([])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => { loadRegionOptions().then(setRegions).catch(() => setRegions([])) }, [])

  const submit = async () => {
    if (busy) return
    if (!regionIds.length) { setErr(w.eRegionsRequired); return }
    setBusy(true); setErr('')
    try {
      await apiFetch(`/workers/${worker.id}/regions`, {
        method: 'PUT',
        body: { regionIds: regionIds.map(Number) },
      })
      toast.success(w.toastRegionsSaved)
      onClose(); onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  const label = 'mb-1 block text-[13px] text-[var(--shell-group-title)]'
  return (
    <Shell title={`${w.regionsTitle} · ${worker.name}`} onClose={onClose}>
      <label className={label}>{w.workerRegion}</label>
      <div className="mb-3">
        <MultiSelect
          values={regionIds}
          options={regions.map((r) => ({ value: String(r.id), label: r.name }))}
          onChange={setRegionIds}
          ariaLabel={w.workerRegion}
          placeholder={w.workerRegion}
          searchPlaceholder={w.searchPlaceholder}
        />
      </div>
      <p className="mb-3 text-xs text-[var(--shell-group-title)]">{w.workerRegionHint}</p>
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{w.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy} onClick={submit}>{w.save}</Button>
      </div>
    </Shell>
  )
}
