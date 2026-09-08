// 积分任务规则 Tab:列表 + 抽屉式新建 + 停用。周期 ONE_TIME/DAILY/MONTHLY。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { listTasks, createTask, disableTask, type LoyTask } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { RuleStatus } from './RuleStatus'
import {
  PageHead, pagerTexts, ErrorBanner, ToolbarButton, FormField, ActionLink,
  DataTable, type ColumnDef,
} from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'

const PERIOD_VALUES = ['ONE_TIME', 'DAILY', 'MONTHLY'] as const

/** 任务周期下拉选项:label 走 i18n,随语言切换。 */
function periodOptions(m: ReturnType<typeof useT>['pages']['marketing']) {
  const labels = { ONE_TIME: m.taskPeriodOnce, DAILY: m.taskPeriodDaily, MONTHLY: m.taskPeriodMonthly }
  return PERIOD_VALUES.map((value) => ({ value, label: labels[value] }))
}

const EMPTY_FORM = { code: '', name: '', points: '', period: 'ONE_TIME' }

export default function TasksTab() {
  const t = useT()
  const m = t.pages.marketing
  const confirmDialog = useConfirm()
  const periodOpts = periodOptions(m)
  const [items, setItems] = useState<LoyTask[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [open, setOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState(EMPTY_FORM)

  const load = () => {
    setError('')
    listTasks()
      .then((d) => setItems(d?.tasks ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const closeForm = () => {
    setOpen(false)
    setForm(EMPTY_FORM)
    setFormError('')
  }

  const submit = async () => {
    if (creating) return
    setFormError('')
    if (!form.code.trim() || !form.name.trim() || !Number(form.points)) {
      setFormError(m.formIncomplete); return
    }
    setCreating(true)
    try {
      await createTask({
        code: form.code.trim(), name: form.name.trim(),
        points: Number(form.points), period: form.period as LoyTask['period'],
      })
      toast.success(m.createdOk)
      closeForm()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setCreating(false)
    }
  }

  const disable = async (id: number, name: string) => {
    if (!(await confirmDialog(m.disableConfirm.replace('{name}', name), { danger: true }))) return
    try { await disableTask(id); toast.success(m.disabledOk); load() } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  const columns: ColumnDef[] = [
    { key: 'code', label: m.taskCode, render: (r) => <span className="font-medium">{String(r.code ?? '—')}</span> },
    { key: 'name', label: m.colName, render: (r) => String(r.name ?? '—') },
    { key: 'points', label: m.taskPoints, render: (r) => String(r.points ?? '—') },
    { key: 'period', label: m.taskPeriod, render: (r) => periodOpts.find((o) => o.value === r.period)?.label ?? String(r.period) },
    { key: 'status', label: m.colStatus, render: (r) => (
      <RuleStatus status={String(r.status)} />
    ) },
    { key: 'op', label: m.colOp, render: (r) => (r.status === 'ENABLED'
      ? <ActionLink onClick={() => disable(Number(r.taskId), String(r.name))} label={m.disable} />
      : null) },
  ]

  const paged = items.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div>
      <PageHead title={m.tabTasks} desc={m.desc} />
      <Card className="p-4">
        <div className="mb-3 flex justify-end">
          <ToolbarButton primary onClick={() => setOpen(true)}>+ {m.create}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <DataTable columns={columns} rows={paged.map((r) => ({ ...r }))} emptyText={m.empty} />
        )}
        {!error && items.length > 0 && (
          <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={items.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(m)} />
          </div>
        )}
      </Card>
      {open && (
        <Drawer title={m.create} onClose={closeForm}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={closeForm}>
                {t.common.confirmDialog.cancel}
              </button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={creating} onClick={submit}>
                {creating ? m.creating : m.create}
              </button>
            </>
          }>
          <div className="grid grid-cols-2 gap-3">
            <FormField label={m.taskCode} required>
              <Input value={form.code} placeholder={m.taskCodePh}
                onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </FormField>
            <FormField label={m.colName} required>
              <Input value={form.name} placeholder={m.taskNamePh}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
            <FormField label={m.taskPoints} required>
              <Input inputMode="numeric" value={form.points} placeholder={m.taskPointsPh}
                onChange={(e) => setForm({ ...form, points: e.target.value })} />
            </FormField>
            <FormField label={m.taskPeriod}>
              <Dropdown value={form.period} options={periodOpts} ariaLabel={m.taskPeriod}
                onChange={(v) => setForm({ ...form, period: v })} />
            </FormField>
          </div>
          {formError && <div className="mt-3"><ErrorBanner message={formError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
