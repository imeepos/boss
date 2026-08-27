// 积分任务规则 Tab:列表 + 抽屉式新建 + 停用。周期 ONE_TIME/DAILY/MONTHLY。
import { useEffect, useState } from 'react'
import { listTasks, createTask, disableTask, type LoyTask } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton, FormField } from '../../../components/business'
import { Dropdown } from '../../../components/Dropdown'
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
  const periodOpts = periodOptions(m)
  const [items, setItems] = useState<LoyTask[]>([])
  const [error, setError] = useState('')
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
      closeForm()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setCreating(false)
    }
  }

  const disable = async (id: number) => {
    try { await disableTask(id); load() } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    }
  }

  return (
    <div>
      <Card className="p-4">
        <div className="mb-3 flex justify-end">
          <ToolbarButton primary onClick={() => setOpen(true)}>+ {m.create}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{m.taskCode}</TableHead>
                <TableHead>{m.colName}</TableHead>
                <TableHead>{m.taskPoints}</TableHead>
                <TableHead>{m.taskPeriod}</TableHead>
                <TableHead>{m.colStatus}</TableHead>
                <TableHead>{m.colOp}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((r) => (
                <TableRow key={r.taskId}>
                  <TableCell className="font-medium">{r.code}</TableCell>
                  <TableCell>{r.name}</TableCell>
                  <TableCell>{r.points}</TableCell>
                  <TableCell>{periodOpts.find((o) => o.value === r.period)?.label ?? r.period}</TableCell>
                  <TableCell>
                    <Badge variant={r.status === 'ENABLED' ? 'success' : 'default'}>{r.status}</Badge>
                  </TableCell>
                  <TableCell>
                    {r.status === 'ENABLED' && (
                      <button className="text-xs text-[var(--color-text-link)] hover:underline"
                        onClick={() => disable(r.taskId)}>{m.disable}</button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={6}><EmptyState text={m.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
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
              <Input value={form.code}
                onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </FormField>
            <FormField label={m.colName} required>
              <Input value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
            <FormField label={m.taskPoints} required>
              <Input inputMode="numeric" value={form.points}
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
