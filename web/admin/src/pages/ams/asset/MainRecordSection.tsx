// 详情抽屉主档字段区:资产编码/标签编号/EPC/类型/型号/批次/部署地址/状态/企业/区域。
// 字段对齐 fields.md §4.1;标签编号/EPC 经 tag_id 反查,型号经 model_id 反查。
import { StatusTag } from '../../../components/StatusTag'
import { useT } from '../../../i18n'
import type { AssetModelRow, AssetRow, TagRow } from '../types'
import { modelLabel } from './logic'

function Item({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 text-[13px]">
      <span className="text-xs text-[var(--shell-group-title)]">{label}</span>
      <span className="truncate text-[var(--shell-content-text)]">{children}</span>
    </div>
  )
}

export function MainRecordSection({ main, tag, model, batch }: {
  main: AssetRow
  tag?: TagRow
  model: AssetModelRow | null
  batch: { id: number; code: string } | null
}) {
  const t = useT()
  const a = t.pages.assetPage
  return (
    <div className="mx-4 mt-4 mb-3 rounded-sm border border-[var(--shell-side-border)] p-3">
      <div className="mb-2 text-xs font-medium text-[var(--shell-group-title)]">{a.mainTitle}</div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
        <Item label={a.dCode}>{main.assetCode}</Item>
        <Item label={a.dTagNo}>{tag?.tagNo || '—'}</Item>
        <Item label={a.dEpc}>{tag?.epcCode || '—'}</Item>
        <Item label={a.dType}>{main.type || '—'}</Item>
        <Item label={a.dModel}>{model ? modelLabel(model) : main.modelId ? '#' + String(main.modelId) : '—'}</Item>
        <Item label={a.dBatch}>{batch ? '#' + String(batch.id) + ' ' + batch.code : '#' + String(main.batchId)}</Item>
        <Item label={a.dAddress}>{main.addressId ? '#' + String(main.addressId) : '—'}</Item>
        <Item label={a.dStatus}><StatusTag domain="asset" value={main.status} /></Item>
        <Item label={a.dEntity}>{main.legalEntityName || '—'}</Item>
        <Item label={a.dRegion}>{main.regionName || '—'}</Item>
      </div>
    </div>
  )
}