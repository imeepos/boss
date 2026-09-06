// 资产列表数据源(P3-T1 服务端分页):offset/limit/status/q 下发,参数变化即触发
// 请求(翻页/条数/筛选共用一条路径);tags 拉一页大 limit 供资产表联表展示标签编号/EPC。
// 删除后刷新当前页;40900 阻断项 message 完整进 error 条。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useT } from '../../../i18n'
import type { AssetRow, TagRow } from '../types'

export interface AssetListParams {
  page: number
  pageSize: number
  status: string
  q: string
}

export function useAssetList(params: AssetListParams) {
  const t = useT()
  const a = t.pages.assetPage
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<AssetRow[]>([])
  const [total, setTotal] = useState(0)
  const [tags, setTags] = useState<TagRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    setError('')
    setBusy(true)
    const query = {
      offset: (params.page - 1) * params.pageSize,
      limit: params.pageSize,
      status: params.status || undefined,
      q: params.q.trim() || undefined,
    }
    Promise.all([
      apiFetch<{ items: AssetRow[]; total: number }>('/assets', { query }),
      apiFetch<{ items: TagRow[] }>('/tags', { query: { limit: 200 } })
        .catch(() => ({ items: [] as TagRow[] })),
    ])
      .then(([d, tg]) => {
        setRows(d?.items ?? [])
        setTotal(d?.total ?? 0)
        setTags(tg?.items ?? [])
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
    // loadFail 文案随语言包加载,非请求参数,不入依赖(挂载后恒稳定)。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.page, params.pageSize, params.status, params.q])

  useEffect(() => { load() }, [load])

  const tagOf = (tagId: number) => tags.find((x) => x.tagId === tagId)

  // 删除:二次确认(仅入库且无引用可删);成功后刷新当前页。
  const delRow = async (r: AssetRow) => {
    if (busy || !(await confirmDialog(a.deleteConfirm, { danger: true }))) return
    try {
      await apiFetch('/assets/' + String(r.assetId), { method: 'DELETE' })
      toast.success(a.deleteOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : a.loadFail)
    }
  }

  return { rows, total, error, busy, load, tagOf, delRow }
}