// 许可单详情:证照档案补录/状态流转/附件引用(P-INFRA-1 W4,000211)。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { AttachmentManager } from '../../../components/AttachmentManager'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'
import { PERMIT_KIND_TEXT, PERMIT_STATUS_TEXT, PERMIT_STATUS_VARIANT, type PermitRow } from './permits'

interface TransitionAction { label: string; to: string; needReason?: boolean; danger?: boolean; useApproval?: boolean }

// 转移表与 terms.md 4 一致;useApproval=批准动作随档案表单带批复号/有效期。
function actionsFor(kind: string, status: string): TransitionAction[] {
  if (kind === 'ROW') {
    if (status === 'NOT_STARTED') return [{ label: '提交申请', to: 'PENDING' }, { label: '标记不适用', to: 'NA' }]
    if (status === 'PENDING') return [{ label: '批准', to: 'APPROVED', useApproval: true }, { label: '驳回', to: 'NOT_STARTED', needReason: true, danger: true }]
    if (status === 'APPROVED') return [{ label: '标记过期', to: 'EXPIRED', danger: true }]
    if (status === 'EXPIRED') return [{ label: '发起过期复验', to: 'PENDING' }]
    if (status === 'NA') return [{ label: '取消不适用', to: 'NOT_STARTED' }]
  }
  if (kind === 'PECE') {
    if (status === 'PENDING_SIGN') return [{ label: '签署', to: 'SIGNED' }, { label: '作废', to: 'NA', needReason: true, danger: true }]
    if (status === 'SIGNED') return [{ label: '盖章', to: 'STAMPED' }, { label: '退回补正', to: 'PENDING_SIGN', needReason: true, danger: true }]
    if (status === 'NA') return [{ label: '恢复待签署', to: 'PENDING_SIGN' }]
  }
  return []
}

export function PermitDetail({ permitId, onChanged }: { permitId: number; onChanged: () => void }) {
  const confirmDialog = useConfirm()
  const [permit, setPermit] = useState<PermitRow | null>(null)
  const [edit, setEdit] = useState({ title: '', approvalNo: '', authority: '', validFrom: '', validUntil: '', facilityCode: '', note: '' })
  const [attachmentIds, setAttachmentIds] = useState<number[]>([])
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    setError('')
    try {
      const p = await apiFetch<PermitRow>('/odn/permits/' + permitId)
      setPermit(p ?? null)
      if (p) {
        setEdit({ title: p.title, approvalNo: p.approvalNo, authority: p.authority, validFrom: p.validFrom ?? '', validUntil: p.validUntil ?? '', facilityCode: p.facilityCode ?? '', note: p.note })
        setAttachmentIds(p.attachmentIds ?? [])
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
      toast.error('许可单加载失败', { description: msg })
    }
  }, [permitId])

  useEffect(() => { void load() }, [load])

  const act = async (fn: () => Promise<unknown>, okMsg: string) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await load(); onChanged() }
    catch (e) {
      const msg = e instanceof Error ? e.message : '操作失败'
      setError(msg)
      toast.error(okMsg + ' 失败', { description: msg })
    } finally { setBusy(false) }
  }

  const saveArchive = () => {
    if (!permit) return
    void act(() => apiFetch('/odn/permits/' + permitId, { method: 'PUT', body: { ...edit, kind: permit.kind, chainId: permit.chainId, attachmentIds } }), '档案要素已保存')
  }

  const transition = async (a: TransitionAction) => {
    if (!permit) return
    if (a.needReason && !reason.trim()) { setError('该操作必填原因'); return }
    const okMsg = PERMIT_STATUS_TEXT[a.to] + ' 已流转'
    const body: Record<string, unknown> = { to: a.to, reason: reason.trim() }
    if (a.useApproval) {
      if (!edit.approvalNo.trim() || !edit.validUntil.trim()) { setError('批准须先填写批复号与有效期止'); return }
      body.approvalNo = edit.approvalNo.trim()
      body.validFrom = edit.validFrom.trim()
      body.validUntil = edit.validUntil.trim()
    }
    if (a.to === 'EXPIRED' || a.needReason) {
      if (!(await confirmDialog('确认执行「' + a.label + '」?', { danger: !!a.danger }))) return
    }
    await act(() => apiFetch('/odn/permits/' + permitId + '/transition', { method: 'POST', body }), okMsg)
    setReason('')
  }

  if (!permit) return error ? <ErrorBanner message={error} /> : <EmptyState text='加载中…' />
  const actions = actionsFor(permit.kind, permit.status)
  const set = (k: string, v: string) => setEdit((m) => ({ ...m, [k]: v }))
  const field = (k: string, label: string, placeholder = '') => <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">{label}</span><Input value={edit[k as keyof typeof edit] ?? ''} placeholder={placeholder} onChange={(e) => set(k, e.target.value)} /></label>
  return <div className="mt-4 space-y-3">
    {error && <ErrorBanner message={error} />}
    <Card className="p-4">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <span className="font-mono text-sm font-semibold">{permit.permitNo}</span>
        <Badge>{PERMIT_KIND_TEXT[permit.kind] ?? permit.kind}</Badge>
        <Badge variant={PERMIT_STATUS_VARIANT[permit.status] ?? 'default'}>{PERMIT_STATUS_TEXT[permit.status] ?? permit.status}</Badge>
        {permit.projectNo && <span className="text-xs opacity-60">项目 {permit.projectNo}</span>}
        {permit.rejectReason && <span className="text-xs text-[var(--color-danger)]">原因:{permit.rejectReason}</span>}
      </div>
      {actions.length > 0 && <div className="flex flex-wrap items-end gap-2">
        {actions.map((a) => <ToolbarButton key={a.to + a.label} disabled={busy} onClick={() => void transition(a)}>{a.label}</ToolbarButton>)}
        {actions.some((a) => a.needReason) && <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">操作原因(驳回/退回/作废必填)</span><Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="原因" /></label>}
      </div>}
    </Card>
    <Card className="p-4">
      <div className="mb-2 text-sm font-semibold">证照档案要素(任何状态可补录)</div>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {field('title', '名称')}
        {field('approvalNo', '批复号')}
        {field('authority', '管辖机构')}
        {field('validFrom', '有效期起', 'YYYY-MM-DD')}
        {field('validUntil', '有效期止', 'YYYY-MM-DD')}
        {field('facilityCode', '关联设施', '如 CLS00001')}
        {field('note', '备注')}
      </div>
      <div className="mt-3 flex justify-end"><ToolbarButton primary disabled={busy} onClick={saveArchive}>{busy ? '保存中…' : '保存档案与附件'}</ToolbarButton></div>
    </Card>
    <Card className="p-4">
      <div className="mb-2 text-sm font-semibold">证照附件(勾选后随档案保存)</div>
      <AttachmentManager uploaderType="account" selectable selectedIds={attachmentIds} onSelectionChange={setAttachmentIds} />
    </Card>
  </div>
}