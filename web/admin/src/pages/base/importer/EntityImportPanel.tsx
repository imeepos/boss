// 业务实体批量导入面板:模板下载(Excel/JSON)+ 文件/附件选择 + 预览 + 逐行调用既有创建端点。
// 与 addr/geo 面板差异:执行为客户端逐行 POST,进度与失败行逐条反馈;401 中止剩余行。
import { useMemo, useRef, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { ApiError } from '../../../api/envelope'
import { ToolbarButton } from '../../../components/business/page-head'
import { Badge } from '../../../components/ui/badge'
import type { Translations } from '../../../i18n/types'
import { MAX_BYTES, PREVIEW_ROWS, parseJson } from './preview'
import { entityTemplateJson, parseEntityRows, MAX_IMPORT_ROWS } from './entityPreview'
import { entityExcelTemplate, isExcelFile, parseEntityExcel, type ExcelParseResult } from './excel'
import { AttachmentPickerDialog } from './AttachmentPickerDialog'
import type { EntityDef } from './entities'

type Text = Translations['pages']['importer']

const AREA_CLS = 'min-h-35 resize-y rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-2 font-mono text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]'
const DROP_CLS = 'cursor-pointer rounded-sm border border-dashed border-[var(--shell-input-border)] bg-[var(--shell-menu-hover-bg)] px-4 py-6 text-center text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)]'
const TH = 'h-9 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const TD = 'h-9 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'

interface RowFailure { row: number; msg: string }

function excelErrorText(r: Extract<ExcelParseResult, { ok: false }>, text: Text): string {
  if (r.reason === 'noSheet') return text.excelNoSheet
  if (r.reason === 'badHeader') return text.excelBadHeader.replace('{sheet}', r.sheet ?? '')
  return text.excelBadRow.replace('{sheet}', r.sheet ?? '').replace('{row}', String(r.row ?? ''))
}

export function EntityImportPanel({ def, text, onImported }: { def: EntityDef; text: Text; onImported: () => void }) {
  const [payload, setPayload] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null)
  const [summary, setSummary] = useState('')
  const [failures, setFailures] = useState<RowFailure[]>([])
  const [pickerOpen, setPickerOpen] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const parsed = useMemo(() => {
    if (!payload.trim()) return null
    const json = parseJson(payload)
    return json.ok ? parseEntityRows(def, json.value) : json
  }, [payload, def])

  const rows = parsed?.ok ? parsed.rows : null
  /** 原始行(未经矫正):失败行导出重试文件的数据源,与 rows 下标一一对齐。 */
  const rawRows = parsed?.ok ? parsed.rawRows : null

  /** 失败行导出重试:原始输入行组装为可直接再导入的 JSON 数组文件。 */
  const exportFailed = () => {
    if (!rawRows || failures.length === 0) return
    const retry = failures.map((f) => rawRows[f.row - 1]).filter((v) => v !== undefined)
    const url = URL.createObjectURL(new Blob([JSON.stringify(retry, null, 2)], { type: 'application/json' }))
    const a = document.createElement('a')
    a.href = url
    a.download = `${def.kind}-import-failed-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  const readFile = (file: File) => {
    if (file.size > MAX_BYTES) { setError(text.fileTooLarge); return }
    if (isExcelFile(file.name, file.type)) {
      file.arrayBuffer()
        .then((buf) => {
          const r = parseEntityExcel(def, buf)
          if (!r.ok) { setError(excelErrorText(r, text)); return }
          setPayload(JSON.stringify(r.value, null, 2))
          setError('')
        })
        .catch(() => setError(text.readFail))
      return
    }
    if (!/\.json$/i.test(file.name) && file.type !== 'application/json') { setError(text.onlyJson); return }
    const reader = new FileReader()
    reader.onload = () => { setPayload(String(reader.result ?? '')); setError('') }
    reader.onerror = () => setError(text.readFail)
    reader.readAsText(file)
  }

  const downloadTemplate = (fmt: 'xlsx' | 'json') => {
    const blob = fmt === 'xlsx'
      ? new Blob([entityExcelTemplate(def)], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
      : new Blob([entityTemplateJson(def)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${def.kind}-import-template.${fmt}`
    a.click()
    URL.revokeObjectURL(url)
  }

  /** 逐行 POST;401(登录失效)中止剩余行,业务失败逐条记录不中断。 */
  const run = async () => {
    if (!rows || busy) return
    setBusy(true)
    setError('')
    setFailures([])
    setSummary('')
    setProgress({ done: 0, total: rows.length })
    const fails: RowFailure[] = []
    let ok = 0
    for (let i = 0; i < rows.length; i++) {
      try {
        await apiFetch(def.endpoint, { method: 'POST', body: rows[i] })
        ok++
      } catch (e) {
        const msg = e instanceof Error ? e.message : text.loadFail
        if (e instanceof ApiError && e.unauthorized) {
          setSummary(text.entityAborted.replace('{done}', String(i)).replace('{total}', String(rows.length)))
          setFailures([...fails])
          setProgress({ done: i, total: rows.length })
          setBusy(false)
          return
        }
        fails.push({ row: i + 1, msg })
      }
      setProgress({ done: i + 1, total: rows.length })
    }
    setSummary(text.entityDone.replace('{ok}', String(ok)).replace('{fail}', String(fails.length)))
    setFailures(fails)
    if (ok > 0) onImported()
    setBusy(false)
  }

  return (
    <div className="mb-6 flex flex-col gap-1.5">
      <div
        className={DROP_CLS}
        role="button"
        tabIndex={0}
        onClick={() => fileRef.current?.click()}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') fileRef.current?.click() }}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => { e.preventDefault(); const f = e.dataTransfer.files?.[0]; if (f) readFile(f) }}
      >
        {text.fileButton} · {text.dropHint}
      </div>
      <input
        ref={fileRef}
        type="file"
        accept=".json,.xlsx,application/json,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        className="hidden"
        onChange={(e) => { const f = e.target.files?.[0]; if (f) readFile(f); e.target.value = '' }}
      />
      <div className="mt-2 flex flex-wrap items-center gap-3">
        <ToolbarButton onClick={() => downloadTemplate('xlsx')}>{text.templateExcel}</ToolbarButton>
        <ToolbarButton onClick={() => downloadTemplate('json')}>{text.template}</ToolbarButton>
        <ToolbarButton onClick={() => setAdvanced((v) => !v)}>{text.pasteToggle}</ToolbarButton>
        <ToolbarButton onClick={() => setPickerOpen(true)}>{text.pickFromAttachments}</ToolbarButton>
        {payload && (
          <ToolbarButton onClick={() => { setPayload(''); setError(''); setSummary(''); setFailures([]); setProgress(null) }}>
            {text.clear}
          </ToolbarButton>
        )}
      </div>
      <AttachmentPickerDialog open={pickerOpen} onClose={() => setPickerOpen(false)} onPick={(f) => readFile(f)} />
      {advanced && (
        <textarea
          className={AREA_CLS}
          value={payload}
          placeholder={text.pastePlaceholder}
          onChange={(e) => setPayload(e.target.value)}
        />
      )}
      {parsed && !parsed.ok && (
        <p className="m-0 text-xs text-[var(--color-danger)]">
          {'reason' in parsed && parsed.reason === 'badRow'
            ? text.reasonEntityBadRow.replace('{row}', String(parsed.row)).replace('{field}', parsed.field ?? '')
            : 'reason' in parsed && parsed.reason === 'tooMany'
              ? text.entityTooMany.replace('{count}', String(parsed.count ?? '')).replace('{max}', String(MAX_IMPORT_ROWS))
              : text.reasonNotArray}
          {'line' in parsed && parsed.line !== undefined ? ` (${text.parseFailAt.replace('{line}', String(parsed.line))})` : ''}
        </p>
      )}
      {rows && (
        <div className="mt-1 rounded-sm border border-[var(--shell-side-border)]">
          <p className="m-0 px-3 pt-2.5 text-xs text-[var(--shell-content-text)]">
            {text.previewOf.replace('{count}', String(rows.length))}
          </p>
          <div className="mt-2 overflow-x-auto">
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr><th key="__no" className={TH}>#</th>{def.columns.map((c) => <th key={c.key} className={TH}>{c.key}</th>)}</tr>
              </thead>
              <tbody>
                {rows.slice(0, PREVIEW_ROWS).map((r, i) => (
                  <tr key={i}>
                    <td key="__no" className={TD}>{i + 1}</td>
                    {def.columns.map((c) => {
                      const v = r[c.key]
                      return <td key={c.key} className={TD}>{v === undefined ? '' : String(v)}</td>
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {rows.length > PREVIEW_ROWS && (
            <p className="m-0 px-3 pb-2.5 text-[11px] text-[var(--shell-group-title)]">
              {text.previewTruncated} ({PREVIEW_ROWS})
            </p>
          )}
        </div>
      )}
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <ToolbarButton primary disabled={!rows || rows.length === 0 || busy} onClick={run}>
          {busy ? text.importing : text.importBtn}
        </ToolbarButton>
        {progress && !busy && <Badge variant={failures.length ? 'warning' : 'success'}>{summary}</Badge>}
        {busy && progress && (
          <span className="text-xs text-[var(--shell-group-title)]">
            {text.entityProgress.replace('{done}', String(progress.done)).replace('{total}', String(progress.total))}
          </span>
        )}
        {error && <span className="text-xs text-[var(--color-danger)]">{error}</span>}
      </div>
      {failures.length > 0 && (
        <div className="mt-1 flex flex-col gap-1.5">
          <ul className="m-0 max-h-40 list-none overflow-y-auto rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] px-3 py-2 text-xs text-[var(--color-danger)]">
            {failures.slice(0, 20).map((f) => (
              <li key={f.row}>{text.entityRowFail.replace('{row}', String(f.row)).replace('{msg}', f.msg)}</li>
            ))}
            {failures.length > 20 && <li>… {text.entityFailMore.replace('{count}', String(failures.length - 20))}</li>}
          </ul>
          <div>
            <ToolbarButton onClick={exportFailed}>{text.entityExportFail}</ToolbarButton>
          </div>
        </div>
      )}
    </div>
  )
}
