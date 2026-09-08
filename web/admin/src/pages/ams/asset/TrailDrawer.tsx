// 资产详情抽屉:顶部关联区块(批次-采购入库单-标签链,data-relations §2.6)+ 状态轨迹/持有台账页签。
// 契约 GET /assets/batches、/procurement/receipts(小表全量,客户端按 batchId 过滤)、
// /assets/:assetId/lifecycle、/assets/:assetId/assignments;页签复用 business TabBar。
import { useEffect, useState, type ReactNode } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { TabBar } from '../../../components/business/tab-bar'
import { useT } from '../../../i18n'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { MainRecordSection } from './MainRecordSection'
import { batchLabel } from './logic'
import { fmtTime } from '../../../lib/format'
import type { AssetModelRow, AssetRow, AssignmentRow, LifecycleRow, TagRow } from '../types'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { TableStateRow } from '../../../components/business'

type BatchRow = { id: number; code: string; name: string }
type ReceiptLite = { id: number; receiptNo: string; orderNo: string; batchId: number; status: string }
type RelTab = 'lifecycle' | 'assignment'

function RelItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 text-[13px]">
      <span className="text-xs text-[var(--shell-group-title)]">{label}</span>
      <span className="truncate text-[var(--shell-content-text)]">{children}</span>
    </div>
  )
}

export function AssetTrailDrawer({
  asset, tag, showMain, onClose,
}: { asset: AssetRow; tag?: TagRow; showMain?: boolean; onClose: () => void }) {
  const t = useT()
  const a = t.pages.assetPage
  const [tab, setTab] = useState<RelTab>('lifecycle')
  const [lifecycle, setLifecycle] = useState<LifecycleRow[] | null>(null)
  const [assignments, setAssignments] = useState<AssignmentRow[] | null>(null)
  const [batch, setBatch] = useState<BatchRow | null>(null)
  const [receipt, setReceipt] = useState<ReceiptLite | null>(null)
  const [error, setError] = useState('')
  const [main, setMain] = useState<AssetRow>(asset)
  const [model, setModel] = useState<AssetModelRow | null>(null)

  useEffect(() => {
    apiFetch<{ items: LifecycleRow[] }>(`/assets/${asset.assetId}/lifecycle`)
      .then((d) => setLifecycle(d?.items ?? []))
      .catch(() => setError(a.loadFail))
    apiFetch<{ items: AssignmentRow[] }>(`/assets/${asset.assetId}/assignments`)
      .then((d) => setAssignments(d?.items ?? []))
      .catch(() => setAssignments([]))
    // 关联链(批次-采购入库单):receipts/batches 为小表全量接口,客户端按 batchId 命中。
    apiFetch<{ items: BatchRow[] }>('/assets/batches')
      .then((d) => setBatch((d?.items ?? []).find((x) => x.id === asset.batchId) ?? null))
      .catch(() => setBatch(null))
    apiFetch<{ items: ReceiptLite[] }>('/procurement/receipts')
      .then((d) => setReceipt((d?.items ?? []).find((x) => x.batchId === asset.batchId) ?? null))
      .catch(() => setReceipt(null))
    // 主档全字段:契约 GET /assets/:assetId;后端未部署时回退列表行数据。
    if (showMain) {
      apiFetch<AssetRow>('/assets/' + String(asset.assetId))
        .then((d) => { if (d) setMain(d) })
        .catch(() => setMain(asset))
      apiFetch<{ items: AssetModelRow[] }>('/asset-models')
        .then((d) => setModel((d?.items ?? []).find((m) => m.id === asset.modelId) ?? null))
        .catch(() => setModel(null))
    }
  }, [asset.assetId, asset.batchId]) // eslint-disable-line react-hooks/exhaustive-deps

  const tabs = [
    { key: 'lifecycle' as RelTab, label: a.lifecycle },
    { key: 'assignment' as RelTab, label: a.assignment },
  ]

  return (
    <Drawer title={`${showMain ? a.detailTitle : a.lifecycleTitle} · ${asset.assetCode}`} onClose={onClose} width={680}
      footer={<ToolbarButton onClick={onClose}>{t.pages.company.cancel}</ToolbarButton>}>
      {/* 关联区块:批次 → 采购入库单 → 采购订单;标签经 /tags 联表传入。 */}
      <div className="mx-4 mt-4 mb-3 rounded-sm border border-[var(--shell-side-border)] p-3">
        <div className="mb-2 text-xs font-medium text-[var(--shell-group-title)]">{a.relTitle}</div>
        <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
          <RelItem label={a.relBatch}>{batchLabel(batch, asset.batchId)}</RelItem>
          <RelItem label={a.relReceipt}>{receipt ? receipt.receiptNo : '—'}</RelItem>
          <RelItem label={a.relOrder}>{receipt?.orderNo || '—'}</RelItem>
          <RelItem label={a.relTag}>{tag ? `${tag.tagNo} · ${tag.epcCode}` : asset.tagId ? `#${asset.tagId}` : '—'}</RelItem>
        </div>
      </div>
      {showMain && <MainRecordSection main={main} tag={tag} model={model} batch={batch} />}
      <TabBar tabs={tabs} value={tab} onChange={setTab} />
      {error && <ErrorBanner message={error} />}
      <div className="px-4 pb-4">
        {tab === 'lifecycle' ? (
          <Table>
            <TableHeader>
              <TableRow>{a.lifecycleColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {(lifecycle ?? []).map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{fmtTime(r.changedAt)}</TableCell>
                  <TableCell><StatusTag domain="asset" value={r.status} /></TableCell>
                  <TableCell>{r.addressName || (r.addressId ? '#' + r.addressId : '—')}</TableCell>
                  <TableCell>{r.workerName || (r.workerId ? '#' + r.workerId : '—')}</TableCell>
                </TableRow>
              ))}
              <TableStateRow colSpan={4} loading={lifecycle === null} text={a.empty} />
            </TableBody>
          </Table>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>{a.assignmentColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {(assignments ?? []).map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.workerName || (r.workerId ? '#' + r.workerId : '—')}</TableCell>
                  <TableCell>{r.addressName || (r.addressId ? '#' + r.addressId : '—')}</TableCell>
                  <TableCell>{r.reason || '—'}</TableCell>
                  <TableCell>{fmtTime(r.effectiveFrom)}</TableCell>
                  <TableCell>{r.effectiveTo ? fmtTime(r.effectiveTo) : '至今'}</TableCell>
                </TableRow>
              ))}
              <TableStateRow colSpan={5} loading={assignments === null} text={a.empty} />
            </TableBody>
          </Table>
        )}
      </div>
    </Drawer>
  )
}