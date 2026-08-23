// 积分任务规则 Tab:列表 + 新建 + 停用。周期 ONE_TIME/DAILY/MONTHLY。
import { useEffect, useState } from 'react'
import { listTasks, createTask, disableTask, type LoyTask } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton, FormField } from '../../../components/business'
import { Dropdown } from '../../../components/Dropdown'

const PERIOD_OPTIONS = [
  { value: 'ONE_TIME', label: '一次性' },
  { value: 'DAILY', label: '每日' },
  { value: 'MONTHLY', label: '每月' },
]

export default function TasksTab() {
  const t = useT()
  const m = t.pages.marketing
  const [items, setItems] = useState<LoyTask[]>([])
  const [error, setError] = useState('')
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState({ code: '', name: '', points: '', period: 'ONE_TIME' })

  const load = () => {
    setError('')
    listTasks()
      .then((d) => setItems(d?.tasks ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

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
      setForm({ code: '', name: '', points: '', period: 'ONE_TIME' })
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
      <Card className="mb-3 p-4">
        <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
          <FormField label={m.taskCode} required>
            <input className="w-full" value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })} />
          </FormField>
          <FormField label={m.colName} required>
            <input className="w-full" value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </FormField>
          <FormField label={m.taskPoints} required>
            <input className="w-full" inputMode="numeric" value={form.points}
              onChange={(e) => setForm({ ...form, points: e.target.value })} />
          </FormField>
          <FormField label={m.taskPeriod}>
            <Dropdown value={form.period} options={PERIOD_OPTIONS} ariaLabel={m.taskPeriod}
              onChange={(v) => setForm({ ...form, period: v })} />
          </FormField>
        </div>
        {formError && <ErrorBanner message={formError} />}
        <div className="flex justify-end">
          <ToolbarButton onClick={submit}>{creating ? m.creating : m.create}</ToolbarButton>
        </div>
      </Card>
      <Card className="p-4">
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
                  <TableCell>{PERIOD_OPTIONS.find((o) => o.value === r.period)?.label ?? r.period}</TableCell>
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
    </div>
  )
}
