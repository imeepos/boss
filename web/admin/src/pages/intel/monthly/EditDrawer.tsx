// 行编辑抽屉:PUT 单行 upsert。字段由 TableMeta 驱动并排除 month/region 与派生列
// (PUT 请求体没有派生字段,前端禁编辑——对应模板灰色「勿填」列)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'
import type { MonthlyRow, TableMeta } from './types'

interface EditDrawerProps {
  meta: TableMeta
  row: MonthlyRow
  onClose: () => void
  onSaved: () => void
}

function numOrNull(v: string): number | null {
  if (v.trim() === '') return null
  const n = Number(v)
  return Number.isFinite(n) && n >= 0 ? Math.trunc(n) : null
}

export function EditDrawer({ meta, row, onClose, onSaved }: EditDrawerProps) {
  const t = useT()
  const m = t.pages.monthlyPage
  const editable = meta.fields.filter((f) => !f.derived && f.key !== 'month' && f.key !== 'region')
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(editable.map((f) => {
      const v = row[f.key]
      return [f.key, typeof v === 'number' ? String(v) : '']
    })))
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const setField = (key: string, v: string) => setValues((prev) => ({ ...prev, [key]: v }))

  const save = async () => {
    if (busy) return
    const valuesBody: Record<string, number> = {}
    for (const f of editable) {
      const n = numOrNull(values[f.key] ?? '')
      if (n !== null) valuesBody[f.key] = n
    }
    setBusy(true)
    setErr('')
    try {
      await apiFetch('/monthly/' + meta.key, {
        method: 'PUT',
        body: { month: row.month, region: row.region, ...valuesBody },
      })
      onSaved()
    } catch (e) {
      setErr(e instanceof Error ? e.message : m.saveFail)
      setBusy(false)
    }
  }

  return (
    <Drawer title={m.editTitle} onClose={onClose} footer={
      <div className="flex items-center gap-3">
        <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => { void save() }}>
          {busy ? m.saving : m.save}
        </button>
        {err && <span className="text-xs text-[var(--color-danger)]">{err}</span>}
      </div>
    }>
      <div className="mb-4 flex gap-6 text-[13px] text-[var(--shell-content-text)]">
        <span>{row.month}</span>
        <span className="font-medium">{row.region}</span>
      </div>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {editable.map((f) => (
          <label key={f.key} className="flex flex-col gap-1 text-xs text-[var(--shell-group-title)]">
            {m[meta.columnsKey][meta.fields.indexOf(f)]}
            <Input
              aria-label={m[meta.columnsKey][meta.fields.indexOf(f)]}
              inputMode="numeric"
              value={values[f.key] ?? ''}
              onChange={(e) => setField(f.key, e.target.value)}
            />
          </label>
        ))}
      </div>
    </Drawer>
  )
}
