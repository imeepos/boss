// CSV 导入导出工具行。导入 multipart 字段名 file;code=42200 表示存在行级失败——
// 已导入计数照常展示(不清空),错误按「行号+原因」逐行列出。导出为裸 CSV(非 envelope),
// 按 Content-Disposition filename 落盘。
import { useRef, useState } from 'react'
import { useT } from '../../../i18n'
import { exportMonthlyCsv, importMonthlyCsv } from './api'
import type { ImportResult, TableKey } from './types'

const BTN = 'h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] text-center hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50'

function fmt(tpl: string, kv: Record<string, string | number>): string {
  return Object.entries(kv).reduce((s, [k, v]) => s.replaceAll('{' + k + '}', String(v)), tpl)
}

interface ImportExportProps {
  table: TableKey
  month: string
  onImported: () => void
}

export function ImportExport({ table, month, onImported }: ImportExportProps) {
  const t = useT()
  const m = t.pages.monthlyPage
  const fileRef = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState<'import' | 'export' | ''>('')
  const [error, setError] = useState('')
  const [partial, setPartial] = useState(false)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [exported, setExported] = useState('')

  const onFile = async (f: File) => {
    setBusy('import')
    setError('')
    setPartial(false)
    setResult(null)
    try {
      const out = await importMonthlyCsv(table, f)
      setResult(out.result)
      setPartial(out.code === 42200)
      onImported()
    } catch (e) {
      setError(e instanceof Error ? e.message : m.importFail)
    } finally {
      setBusy('')
    }
  }

  const doExport = async () => {
    setBusy('export')
    setError('')
    setExported('')
    try {
      setExported(await exportMonthlyCsv(table, month))
    } catch (e) {
      setError(e instanceof Error ? e.message : m.exportFail)
    } finally {
      setBusy('')
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-3">
        <button type="button" className={BTN} disabled={busy !== ''} onClick={() => fileRef.current?.click()}>
          {busy === 'import' ? m.importing : m.import}
        </button>
        <button type="button" className={BTN} disabled={busy !== ''} onClick={() => { void doExport() }}>
          {busy === 'export' ? m.exporting : m.export}
        </button>
        {result && !partial && (
          <span className="text-xs text-[var(--shell-group-title)]">
            {fmt(m.importSummary, { total: result.total, imported: result.imported, failed: result.failed })}
          </span>
        )}
        {result && partial && (
          <span className="text-xs text-[var(--color-danger)]">
            {fmt(m.importPartial, { imported: result.imported, failed: result.failed })}
          </span>
        )}
        {exported && <span className="text-xs text-[var(--shell-group-title)]">{fmt(m.exportSaved, { file: exported })}</span>}
        {error && <span className="text-xs text-[var(--color-danger)]">{error}</span>}
      </div>
      <input
        ref={fileRef}
        type="file"
        accept=".csv,text/csv"
        className="hidden"
        onChange={(e) => { const f = e.target.files?.[0]; if (f) void onFile(f); e.target.value = '' }}
      />
      {result && partial && result.errors && result.errors.length > 0 && (
        <ul className="m-0 max-h-40 list-disc overflow-y-auto rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-6 py-2 text-xs text-[var(--color-danger)]">
          {result.errors.map((e) => (
            <li key={e.line}>{fmt(m.errorLine, { line: e.line, reason: e.reason })}</li>
          ))}
        </ul>
      )}
    </div>
  )
}
