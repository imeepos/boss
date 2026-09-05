// 业务实体导入逻辑 hook:文件读取/模板下载/解析去重/逐行执行/结果登记,自 EntityImportPanel 拆出(行数红线)。
import { useEffect, useMemo, useRef, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { ApiError } from '../../../api/envelope'
import type { Translations } from '../../../i18n/types'
import { MAX_BYTES, parseJson } from './preview'
import { entityTemplateJson, parseEntityRows, splitQueryRow, dedupeIndexes, MAX_IMPORT_ROWS } from './entityPreview'
import { entityExcelTemplate, isExcelFile, parseEntityExcel, type ExcelParseResult } from './excel'
import type { EntityDef } from './entities'

type Text = Translations['pages']['importer']

function importClientKey(kind: string): string {
  return `${kind}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

interface RowFailure { row: number; msg: string }

function excelErrorText(r: Extract<ExcelParseResult, { ok: false }>, text: Text): string {
  if (r.reason === 'noSheet') return text.excelNoSheet
  if (r.reason === 'badHeader') return text.excelBadHeader.replace('{sheet}', r.sheet ?? '')
  return text.excelBadRow.replace('{sheet}', r.sheet ?? '').replace('{row}', String(r.row ?? ''))
}

export function useEntityImport({ def, noPerm, text, onImported }: {
  def: EntityDef
  /** 权限前置:当前账号缺创建端点菜单权限时禁止执行。 */
  noPerm?: boolean
  text: Text
  onImported: () => void
}) {
  const [payload, setPayload] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null)
  const [summary, setSummary] = useState('')
  const [failures, setFailures] = useState<RowFailure[]>([])
  const [pickerOpen, setPickerOpen] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  /** 单次行数上限:业务参数 importer.maxRows 覆盖,缺省 500(读取失败不阻断)。 */
  const [maxRows, setMaxRows] = useState(MAX_IMPORT_ROWS)
  const clientKeyRef = useRef('')
  const [maxRowsFallback, setMaxRowsFallback] = useState(false)
  const [existingLoadFailed, setExistingLoadFailed] = useState(false)
  /** 登记失败后保留的待登记统计(用同一 clientKey 支持重试,不重复执行创建行)。 */
  const [pendingTask, setPendingTask] = useState<{ total: number; imported: number; failed: number; skipped: number } | null>(null)

  useEffect(() => {
    let alive = true
    apiFetch<{ items: Array<{ key: string; value: string }> }>('/params')
      .then((d) => {
        const it = d?.items?.find((p) => p.key === 'importer.maxRows')
        if (!it || !alive) return
        const n = Number(JSON.parse(it.value))
        if (Number.isInteger(n) && n > 0) setMaxRows(n)
         else setMaxRowsFallback(true)
      })
      .catch(() => { if (alive) setMaxRowsFallback(true) })
    return () => { alive = false }
  }, [])

  const parsed = useMemo(() => {
    if (!payload.trim()) return null
    const json = parseJson(payload)
    return json.ok ? parseEntityRows(def, json.value, maxRows) : json
  }, [payload, def, maxRows])

  const rows = parsed?.ok ? parsed.rows : null
  /** 原始行(未经矫正):失败行导出重试文件的数据源,与 rows 下标一一对齐。 */
  const rawRows = parsed?.ok ? parsed.rawRows : null
  /** 现有数据(去重数据源):listEndpoint 拉取,失败不阻断。 */
  const [existing, setExisting] = useState<unknown[] | null>(null)
  useEffect(() => {
    let alive = true
    setExisting(null)
    setExistingLoadFailed(false)
    if (!def.listEndpoint) return
    apiFetch<unknown[] | { items: unknown[] }>(def.listEndpoint)
      .then((d) => {
        if (!alive) return
        setExisting(Array.isArray(d) ? d : d?.items ?? [])
      })
      .catch(() => { if (alive) setExistingLoadFailed(true) })
    return () => { alive = false }
  }, [def])
  /** 去重:文件内先到先得 + 与现有数据比对,行号集合为跳过项。 */
  const dedupe = useMemo(
    () => (rows ? dedupeIndexes(def, rows, existing) : { skip: new Set<number>(), skipped: 0 }),
    [def, rows, existing],
  )

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

  /** 导入结果登记(POST /import-tasks):结果可追溯;登记失败保留待登记统计供同 key 重试。 */
  const registerTask = async (total: number, imported: number, failed: number, skipped: number): Promise<boolean> => {
    try {
      await apiFetch('/import-tasks', {
        method: 'POST',
        body: { kind: `entity:${def.kind}`, total, imported, failed, skipped, clientKey: clientKeyRef.current },
      })
      setPendingTask(null)
      return true
    } catch (e: unknown) {
      console.warn('import-task register failed', e)
      setError(text.taskRegisterFail)
      setPendingTask({ total, imported, failed, skipped })
      return false
    }
  }

  /** 登记重试:只重发登记端点,复用本次 clientKey,不重复执行业务创建行。 */
  const retryRegister = async () => {
    if (!pendingTask || busy) return
    setBusy(true)
    setError('')
    const okReg = await registerTask(pendingTask.total, pendingTask.imported, pendingTask.failed, pendingTask.skipped)
    if (okReg) onImported()
    setBusy(false)
  }

  /** 逐行 POST;401(登录失效)中止剩余行,业务失败逐条记录不中断。 */
  const run = async () => {
    if (!rows || busy || noPerm) return
    setBusy(true)
    setError('')
    setFailures([])
    setSummary('')
    clientKeyRef.current = importClientKey(def.kind)
    const total = rows.length - dedupe.skipped
    setProgress({ done: 0, total })
    const fails: RowFailure[] = []
    let ok = 0
    for (let i = 0; i < rows.length; i++) {
      if (dedupe.skip.has(i)) continue
      try {
        const { query, body } = splitQueryRow(def, rows[i])
        await apiFetch(def.endpoint, { method: 'POST', query, body })
        ok++
      } catch (e) {
        const msg = e instanceof Error ? e.message : text.loadFail
        if (e instanceof ApiError && e.unauthorized) {
          setSummary(text.entityAborted.replace('{done}', String(ok + fails.length)).replace('{total}', String(rows.length))
            + (total - ok - fails.length > 0 ? ' · ' + text.entityUnprocessed.replace('{count}', String(total - ok - fails.length)) : ''))
          setFailures([...fails])
          setProgress({ done: ok + fails.length, total })
          await registerTask(rows.length, ok, fails.length, dedupe.skipped)
          if (ok > 0 || fails.length > 0) onImported()
          setBusy(false)
          return
        }
        fails.push({ row: i + 1, msg })
      }
      setProgress({ done: ok + fails.length, total })
    }
    setSummary(text.entityDone.replace('{ok}', String(ok)).replace('{fail}', String(fails.length))
      + (dedupe.skipped > 0 ? ' · ' + text.entitySkipped.replace('{count}', String(dedupe.skipped)) : ''))
    setFailures(fails)
    await registerTask(rows.length, ok, fails.length, dedupe.skipped)
    if (ok > 0 || fails.length > 0) onImported()
    setBusy(false)
  }
  return {
    payload, setPayload, advanced, setAdvanced, busy, error, progress, setProgress, summary,
    failures, setFailures, pickerOpen, setPickerOpen, fileRef, parsed, rows, dedupe, maxRows,
    maxRowsFallback, existingLoadFailed, pendingTask, run, readFile,
    downloadTemplate, exportFailed, retryRegister, setError, setSummary,
  }
}
