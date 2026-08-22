// 员工管理(partner_admin):本企业员工列表 + 新建(partner_staff)+ 启用/停用。
import { useCallback, useEffect, useState } from 'react'
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

const emptyDraft = { username: '', password: '', realName: '', phone: '' }

export default function PartnerStaffPage() {
  const t = useT()
  const [items, setItems] = useState<PartnerStaff[]>([])
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState(emptyDraft)
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState('')

  const load = useCallback(() => {
    setError('')
    listPartnerStaff()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.partnerStaff.loadFail))
  }, [t]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(load, [load])

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await createPartnerStaff(draft)
      setOpen(false)
      setDraft(emptyDraft)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.partnerStaff.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const toggle = async (r: PartnerStaff) => {
    await setPartnerStaffStatus(r.id, r.status === 1 ? 0 : 1)
    load()
  }

  return (
    <div>
      <PageHead title={t.pages.partnerStaff.title} desc={t.pages.partnerStaff.desc} />
      <div className="mb-3 flex items-center">
        <div className="flex-1" />
        <ToolbarButton primary onClick={() => setOpen(true)}>{t.pages.partnerStaff.add}</ToolbarButton>
      </div>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.pages.partnerStaff.colUsername}</TableHead>
                <TableHead>{t.pages.partnerStaff.colRealName}</TableHead>
                <TableHead>{t.pages.partnerStaff.colPhone}</TableHead>
                <TableHead>{t.pages.partnerStaff.colRole}</TableHead>
                <TableHead>{t.pages.partnerStaff.colStatus}</TableHead>
                <TableHead>{t.pages.partnerStaff.colCreatedAt}</TableHead>
                <TableHead>{t.pages.partnerStaff.colOp}</TableHead>
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
                      ? t.pages.partnerStaff.roleAdmin
                      : t.pages.partnerStaff.roleStaff}
                  </TableCell>
                  <TableCell>
                    <Badge variant={r.status === 1 ? 'success' : 'default'}>
                      {r.status === 1 ? t.pages.partnerStaff.enabled : t.pages.partnerStaff.disabled}
                    </Badge>
                  </TableCell>
                  <TableCell className="whitespace-nowrap">{r.createdAt.slice(0, 10)}</TableCell>
                  <TableCell>
                    {r.roleCode === 'partner_admin' ? (
                      <span className="text-xs text-[var(--shell-crumb-text)]">-</span>
                    ) : (
                      <button
                        className="border-none bg-none px-0 text-xs text-[var(--color-text-link)] cursor-pointer hover:underline"
                        onClick={() => toggle(r)}
                      >
                        {r.status === 1 ? t.pages.partnerStaff.disable : t.pages.partnerStaff.enable}
                      </button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={7}><EmptyState text={t.pages.partnerStaff.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>

      {open && (
        <Drawer
          title={t.pages.partnerStaff.add}
          onClose={() => setOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setOpen(false)}>{t.pages.partnerStaff.cancel}</ToolbarButton>
              <ToolbarButton primary disabled={busy} onClick={submit}>{t.pages.partnerStaff.create}</ToolbarButton>
            </>
          }
        >
          <div className="flex flex-col gap-1">
            <label className="text-xs text-[var(--shell-content-text)]">{t.pages.partnerStaff.colUsername}</label>
            <Input value={draft.username} onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
            <label className="mt-2 text-xs text-[var(--shell-content-text)]">{t.pages.partnerStaff.password}</label>
            <Input type="password" value={draft.password} onChange={(e) => setDraft({ ...draft, password: e.target.value })} />
            <label className="mt-2 text-xs text-[var(--shell-content-text)]">{t.pages.partnerStaff.colRealName}</label>
            <Input value={draft.realName} onChange={(e) => setDraft({ ...draft, realName: e.target.value })} />
            <label className="mt-2 text-xs text-[var(--shell-content-text)]">{t.pages.partnerStaff.colPhone}</label>
            <Input value={draft.phone} onChange={(e) => setDraft({ ...draft, phone: e.target.value })} />
            {formError && <p className="m-0 mt-3 text-xs text-[var(--color-danger)]">{formError}</p>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
