// 员工管理(partner_admin):本企业员工列表 + 新建(partner_staff)+ 启用/停用。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  listPartnerStaff, createPartnerStaff, setPartnerStaffStatus, type PartnerStaff,
} from '../../../api/partner'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Drawer } from '../../../components/Drawer'
import { useConfirm } from '../../../components/ConfirmDialog'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'

const emptyDraft = { username: '', password: '', realName: '', phone: '' }

export default function PartnerStaffPage() {
  const t = useT()
  const p = t.pages.partnerStaff
  const confirmDialog = useConfirm()
  const [items, setItems] = useState<PartnerStaff[]>([])
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(emptyDraft)
  const [submit, setSubmit] = useState<SubmitState>('idle')
  const [togglingId, setTogglingId] = useState<number | null>(null)

  const load = useCallback(() => {
    setError('')
    listPartnerStaff()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
  }, [p.loadFail])

  useEffect(load, [load])

  const submitLabels: Record<SubmitState, string> = {
    idle: p.create, loading: t.common.loading, success: p.create, failed: p.create,
  }

  const onSubmit = async () => {
    if (submit === 'loading') return
    setSubmit('loading')
    try {
      await createPartnerStaff(draft)
      setSubmit('success')
      toast.success(p.create)
      setOpen(false)
      setDraft(emptyDraft)
      load()
    } catch (e) {
      setSubmit('failed')
      toast.error(p.create, { description: e instanceof Error ? e.message : undefined })
    } finally {
      setTimeout(() => setSubmit('idle'), 1500)
    }
  }

  const toggle = async (r: PartnerStaff) => {
    if (togglingId) return
    const nextStatus = r.status === 1 ? 0 : 1
    if (!(await confirmDialog(r.status === 1 ? p.disableConfirm.replace('{name}', r.realName) : p.enableConfirm.replace('{name}', r.realName), { danger: r.status === 1 }))) return
    setTogglingId(r.id)
    setPartnerStaffStatus(r.id, nextStatus)
      .then(() => { toast.success(nextStatus === 1 ? p.enable : p.disable); load() })
      .catch((e) => toast.error(nextStatus === 1 ? p.enable : p.disable, { description: e instanceof Error ? e.message : undefined }))
      .finally(() => setTogglingId(null))
  }

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="mb-3 flex items-center">
        <div className="flex-1" />
        <ToolbarButton primary onClick={() => setOpen(true)}>{p.add}</ToolbarButton>
      </div>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{p.colUsername}</TableHead>
                <TableHead>{p.colRealName}</TableHead>
                <TableHead>{p.colPhone}</TableHead>
                <TableHead>{p.colRole}</TableHead>
                <TableHead>{p.colStatus}</TableHead>
                <TableHead>{p.colCreatedAt}</TableHead>
                <TableHead>{p.colOp}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-medium">{r.username}</TableCell>
                  <TableCell>{r.realName}</TableCell>
                  <TableCell>{r.phone || '-'}</TableCell>
                  <TableCell>
                    {r.roleCode === 'partner_admin'
                      ? p.roleAdmin
                      : p.roleStaff}
                  </TableCell>
                  <TableCell>
                    <Badge variant={r.status === 1 ? 'success' : 'default'}>
                      {r.status === 1 ? p.enabled : p.disabled}
                    </Badge>
                  </TableCell>
                  <TableCell className="whitespace-nowrap">{r.createdAt.slice(0, 10)}</TableCell>
                  <TableCell>
                    {r.roleCode === 'partner_admin' ? (
                      <span className="text-xs text-[var(--shell-crumb-text)]">-</span>
                    ) : (
                      <button
                        className="border-none bg-none px-0 text-xs text-[var(--color-text-link)] cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50"
                        disabled={togglingId !== null}
                        onClick={() => toggle(r)}
                      >
                        {togglingId === r.id ? t.common.loading : r.status === 1 ? p.disable : p.enable}
                      </button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={7}><EmptyState text={p.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>

      {open && (
        <Drawer
          title={p.add}
          onClose={() => { if (submit !== 'loading') { setOpen(false); setDraft(emptyDraft) } }}
          footer={
            <>
              <ToolbarButton onClick={() => { setOpen(false); setDraft(emptyDraft) }}>{p.cancel}</ToolbarButton>
              <SubmitButton state={submit} labels={submitLabels} onClick={onSubmit} disabled={!draft.username || !draft.password || !draft.realName} />
            </>
          }
        >
          <div className="flex flex-col gap-1">
            <label className="text-xs text-[var(--shell-content-text)]">{p.colUsername}<span className="text-[var(--color-danger)]"> *</span></label>
            <Input value={draft.username} onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
            <label className="mt-2 text-xs text-[var(--shell-content-text)]">{p.password}<span className="text-[var(--color-danger)]"> *</span></label>
            <Input type="password" autoComplete="new-password" value={draft.password} onChange={(e) => setDraft({ ...draft, password: e.target.value })} />
            <label className="mt-2 text-xs text-[var(--shell-content-text)]">{p.colRealName}<span className="text-[var(--color-danger)]"> *</span></label>
            <Input value={draft.realName} onChange={(e) => setDraft({ ...draft, realName: e.target.value })} />
            <label className="mt-2 text-xs text-[var(--shell-content-text)]">{p.colPhone}</label>
            <Input value={draft.phone} onChange={(e) => setDraft({ ...draft, phone: e.target.value })} />
          </div>
        </Drawer>
      )}
    </div>
  )
}
