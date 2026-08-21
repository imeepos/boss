// 单个导入面板:文件选择/拖拽为主通道,粘贴 JSON 为高级模式;解析即预览,确认后提交。
// 修复项:地址导入包 {rows} 信封、geo 结果按 countries/subdivisions 计数、后端错误信息透传。
import { useMemo, useRef, useState, type DragEvent } from 'react'
import { apiFetch } from '../../../api/client'
import { ToolbarButton } from '../../../components/business/page-head'
import { Badge } from '../../../components/ui/badge'
import type { Translations } from '../../../i18n/types'
import {
  MAX_BYTES, PREVIEW_ROWS, buildPreview, parseJson, resultCount, templateJson,
  type ImportKind, type PreviewResult,
} from './preview'
import { excelTemplate, isExcelFile, parseExcel, type ExcelParseResult } from './excel'

/** Excel 解析错误 → i18n 文案(sheet/row 定位透传)。 */
function excelErrorText(r: Extract<ExcelParseResult, { ok: false }>, text: Text): string {
  if (r.reason === 'noSheet') return text.excelNoSheet
  if (r.reason === 'badHeader') return text.excelBadHeader.replace('{sheet}', r.sheet ?? '')
  return text.excelBadRow.replace('{sheet}', r.sheet ?? '').replace('{row}', String(r.row ?? ''))
}

const AREA_CLS = 'min-h-35 resize-y rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-2 font-mono text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]'
const DROP_CLS = 'cursor-pointer rounded-sm border border-dashed border-[var(--shell-input-border)] bg-[var(--shell-menu-hover-bg)] px-4 py-6 text-center text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)]'
const TH = 'h-9 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const TD = 'h-9 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'

type Text = Translations['pages']['importer']

export function ImportPanel({ kind, title, hint, endpoint, text, onImported }: {
  kind: ImportKind
  title: string
  hint: string
  endpoint: string
  text: Text
  onImported: () => void
}) {
  const [payload, setPayload] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState('')
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  const parsed = useMemo(() => (payload.trim() ? parseJson(payload) : null), [payload])
  const preview: PreviewResult | null = parsed?.ok ? buildPreview(kind, parsed.value) : null

  const readFile = (file: File) => {
    if (file.size > MAX_BYTES) {
      setError(text.fileTooLarge)
      return
    }
    if (isExcelFile(file.name, file.type)) {
      // Excel 通道:解析为 JSON 通道同构载荷后走统一预览/提交链路。
      file.arrayBuffer()
        .then((buf) => {
          const r = parseExcel(kind, buf)
          if (!r.ok) {
            setError(excelErrorText(r, text))
            return
          }
          setPayload(JSON.stringify(r.value, null, 2))
          setResult('')
          setError('')
        })
        .catch(() => setError(text.readFail))
      return
    }
    if (!/\.json$/i.test(file.name) && file.type !== 'application/json') {
      setError(text.onlyJson)
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      setPayload(String(reader.result ?? ''))
      setResult('')
      setError('')
    }
    reader.onerror = () => setError(text.readFail)
    reader.readAsText(file)
  }

  const onDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    const f = e.dataTransfer.files?.[0]
    if (f) readFile(f)
  }

  const downloadTemplate = (fmt: 'xlsx' | 'json') => {
    const blob = fmt === 'xlsx'
      ? new Blob([excelTemplate(kind)], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
      : new Blob([templateJson(kind)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${kind === 'addr' ? 'address-import' : 'geo-import'}-template.${fmt}`
    a.click()
    URL.revokeObjectURL(url)
  }

  const run = async () => {
    if (!preview?.ok || busy) return
    setBusy(true)
    setError('')
    setResult('')
    try {
      const res = await apiFetch<Record<string, unknown>>(endpoint, {
        method: 'POST',
        body: preview.model.body,
      })
      setResult(resultText(res))
      onImported()
    } catch (e) {
      setError(e instanceof Error ? e.message : text.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const resultText = (res: unknown): string => {
    const c = resultCount(kind, res)
    if (kind === 'geo') {
      return text.importedGeo
        .replace('{countries}', String(c.countries ?? 0))
        .replace('{subdivisions}', String(c.subdivisions ?? 0))
    }
    return text.imported.replace('{count}', String(c.total ?? 0))
  }

  return (
    <div className="mb-6 flex flex-col gap-1.5">
      <label className="text-sm text-[var(--shell-content-text)]">{title}</label>
      <p className="m-0 mb-1 text-xs text-[var(--shell-input-placeholder)]">{hint}</p>
      <div
        className={DROP_CLS}
        role="button"
        tabIndex={0}
        onClick={() => fileRef.current?.click()}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') fileRef.current?.click() }}
        onDragOver={(e) => e.preventDefault()}
        onDrop={onDrop}
      >
        {text.fileButton} · {text.dropHint}
      </div>
      <input
        ref={fileRef}
        type="file"
        accept=".json,.xlsx,application/json,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) readFile(f)
          e.target.value = ''
        }}
      />
      <div className="mt-2 flex items-center gap-3">
        <ToolbarButton onClick={() => downloadTemplate('xlsx')}>{text.templateExcel}</ToolbarButton>
        <ToolbarButton onClick={() => downloadTemplate('json')}>{text.template}</ToolbarButton>
        <ToolbarButton onClick={() => setAdvanced((v) => !v)}>{text.pasteToggle}</ToolbarButton>
        {payload && (
          <ToolbarButton onClick={() => { setPayload(''); setResult(''); setError('') }}>
            {text.clear}
          </ToolbarButton>
        )}
      </div>
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
          {parsed.line !== undefined ? text.parseFailAt.replace('{line}', String(parsed.line)) : text.parseFail}
        </p>
      )}
      {parsed?.ok && preview && !preview.ok && (
        <p className="m-0 text-xs text-[var(--color-danger)]">
          {preview.reason === 'badRow'
            ? text.reasonBadRow.replace('{row}', String(preview.row))
            : preview.reason === 'notArray' ? text.reasonNotArray : text.reasonNotObject}
        </p>
      )}
      {preview?.ok && (
        <div className="mt-1 rounded-sm border border-[var(--shell-side-border)]">
          <p className="m-0 px-3 pt-2.5 text-xs text-[var(--shell-content-text)]">
            {text.previewOf.replace('{count}', String(preview.model.rowCount))}
          </p>
          {kind === 'addr' ? (
            <div className="mt-2 overflow-x-auto">
              <table className="w-full border-collapse text-[13px]">
                <thead>
                  <tr>{text.addrColumns.map((c) => <th key={c} className={TH}>{c}</th>)}</tr>
                </thead>
                <tbody>
                  {preview.model.rows.map((cells, i) => (
                    <tr key={i}>{cells.map((v, j) => <td key={j} className={TD}>{v}</td>)}</tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <ul className="m-0 mt-1 list-none px-3 pb-2 text-xs text-[var(--shell-content-text)]">
              {preview.model.sections.map((s) => (
                <li key={s.labelIndex}>{text.geoSections[s.labelIndex]}: {s.count}</li>
              ))}
            </ul>
          )}
          {preview.model.truncated && (
            <p className="m-0 px-3 pb-2.5 text-[11px] text-[var(--shell-group-title)]">
              {text.previewTruncated} ({PREVIEW_ROWS})
            </p>
          )}
        </div>
      )}
      <div className="mt-3 flex items-center gap-3">
        <ToolbarButton primary disabled={!preview?.ok || busy} onClick={run}>
          {busy ? text.importing : text.importBtn}
        </ToolbarButton>
        {result && <Badge variant="success">{result}</Badge>}
        {error && <span className="text-xs text-[var(--color-danger)]">{error}</span>}
      </div>
    </div>
  )
}
