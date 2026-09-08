// 企业员工面板(000173):公司管理页按企业维度维护员工登录信息。
// 列表列名对齐 fields.md 1.1(工号/登录名/姓名/手机/角色/状态);启停走确认弹窗。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { toast } from 'sonner'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Button } from '../../../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner } from '../../../components/business/page-head'
import { EntityStaffDialogs, type EntityStaffDialogMode } from './EntityStaffDialogs'
import { compact } from '../../boss/worker/TeamDialogs'

interface StaffRow {
  id: number
  username: string
  staffNo: string
  realName: string
  phone: string
  roleCode: string
  status: number
}

// EntityStaffPanel 员工抽屉:列表 + 录入/重置密码对话框 + 启停。
export function EntityStaffPanel({ entityId, entityName, onClose }: { entityId: number; entityName: string; onClose: () => void }) {
  const t = useT().pages.company.staff
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<StaffRow[]>([])
  const [error, setError] = useState('')
  const [dialog, setDialog] = useState<EntityStaffDialogMode>(null)

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: StaffRow[] }>(`/legal-entities/${entityId}/staff`)
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.loadFail))
  }, [entityId, t.loadFail])
  useEffect(load, [load]) // eslint-disable-line react-hooks/exhaustive-deps

  const toggleStatus = async (r: StaffRow) => {
    const next = r.status === 1 ? 0 : 1
    if (!(await confirmDialog(next === 0 ? t.disableConfirm.replace('{name}', r.realName) : t.enableConfirm.replace('{name}', r.realName), { danger: next === 0 }))) return
    try {
      await apiFetch(`/legal-entities/${entityId}/staff/${r.id}/status`, { method: 'PUT', body: { status: next } })
      toast.success(t.statusOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t.actionFail)
    }
  }

  const linkBtn = 'cursor-pointer border-none bg-transparent p-0 text-[13px] text-[var(--color-text-link)] hover:underline'
  const cols = t.columns

  return (
    <Drawer title={`${t.panelTitle} · ${entityName}`} onClose={onClose}
      footer={
        <>
          <Button variant="outline" size="sm" className={compact} onClick={onClose}>{t.cancel}</Button>
          <Button size="sm" className={compact} onClick={() => setDialog({ type: 'create', entityId })}>{t.add}</Button>
        </>
      }>
      {error && <ErrorBanner message={error} />}
      <Table>
        <TableHeader>
          <TableRow>
            {cols.map((c) => <TableHead key={c}>{c}</TableHead>)}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell>{r.staffNo || '-'}</TableCell>
              <TableCell>{r.username}</TableCell>
              <TableCell>{r.realName}</TableCell>
              <TableCell>{r.phone || '-'}</TableCell>
              <TableCell>{r.roleCode === 'partner_admin' ? t.roleAdmin : t.roleStaff}</TableCell>
              <TableCell>
                <span className={r.status === 1 ? 'text-[var(--color-success)]' : 'text-[var(--shell-group-title)]'}>
                  {r.status === 1 ? t.statusOn : t.statusOff}
                </span>
              </TableCell>
              <TableCell>
                <span className="inline-flex items-center gap-2">
                  <button className={linkBtn} onClick={() => setDialog({ type: 'resetPwd', entityId, accountId: r.id, name: r.realName })}>{t.resetPwd}</button>
                  <span className="text-[var(--shell-side-border)]">|</span>
                  <button className={linkBtn} onClick={() => toggleStatus(r)}>{r.status === 1 ? t.disable : t.enable}</button>
                </span>
              </TableCell>
            </TableRow>
          ))}
          {!rows.length && !error && (
            <TableRow><TableCell colSpan={cols.length} className="text-center text-[var(--shell-group-title)]">{t.empty}</TableCell></TableRow>
          )}
        </TableBody>
      </Table>
      <EntityStaffDialogs mode={dialog} onClose={() => setDialog(null)} onDone={load} />
    </Drawer>
  )
}
