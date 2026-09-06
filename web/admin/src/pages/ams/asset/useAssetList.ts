// 资产列表数据源:清单/标签加载与删除(40900 阻断项 message 完整进 error 条)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useT } from '../../../i18n'
import type { AssetRow, TagRow } from '../types'

export function useAssetList() {
  const t = useT()
  const a = t.pages.assetPage
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<AssetRow[]>([])
  const [tags, setTags] = useState<TagRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: AssetRow[] }>('/assets'),
      apiFetch<{ items: TagRow[] }>('/tags').catch(() => ({ items: [] as TagRow[] })),
    ])
      .then(([d, tg]) => { setRows(d?.items ?? []); setTags(tg?.items ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const tagOf = (tagId: number) => tags.find((x) => x.tagId === tagId)

  // 删除:二次确认(仅入库且无引用可删);40900 时服务端 message(阻断项清单)完整展示。
  const delRow = async (r: AssetRow) => {
    if (busy || !(await confirmDialog(a.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch('/assets/' + String(r.assetId), { method: 'DELETE' })
      toast.success(a.deleteOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : a.loadFail)
    } finally {
      setBusy(false)
    }
  }

  return { rows, error, busy, load, tagOf, delRow }
}