// 业务实体批量导入面板:模板下载(Excel/JSON)+ 文件/附件选择 + 预览 + 逐行调用既有创建端点。
// 与 addr/geo 面板差异:执行为客户端逐行 POST,进度与失败行逐条反馈;401 中止剩余行。
// 视图层:拖放/工具栏/预览表/进度与失败清单,逻辑见 useEntityImport.ts。
import { ToolbarButton, ErrorBanner } from '../../../components/business/page-head'
import { Badge } from '../../../components/ui/badge'
import { Textarea } from '../../../components/ui/textarea'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import type { Translations } from '../../../i18n/types'
import { PREVIEW_ROWS } from './preview'
import { AttachmentPickerDialog } from './AttachmentPickerDialog'
import { useEntityImport } from './useEntityImport'
import type { EntityDef } from './entities'

const DROP_CLS = 'cursor-pointer rounded-sm border border-dashed border-[var(--shell-input-border)] bg-[var(--shell-menu-hover-bg)] px-4 py-6 text-center text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)]'

export function EntityImportPanel({ def, noPerm, text, onImported }: {
  def: EntityDef
  /** 权限前置:当前账号缺创建端点菜单权限时禁止执行。 */
  noPerm?: boolean
  text: Translations['pages']['importer']
  onImported: () => void
}) {
  const {
    payload, setPayload, advanced, setAdvanced, busy, error, progress, summary,
    failures, pickerOpen, setPickerOpen, fileRef, parsed, rows, dedupe, maxRows,
    maxRowsFallback, existingLoadFailed, pendingTask, run, readFile,
    downloadTemplate, exportFailed, retryRegister, setFailures, setProgress,
    setError, setSummary,
  } = useEntityImport({ def, noPerm, text, onImported })
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
        <Textarea
          className="min-h-35 font-mono text-xs"
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
              ? text.entityTooMany.replace('{count}', String(parsed.count ?? '')).replace('{max}', String(maxRows))
              : text.reasonNotArray}
          {'line' in parsed && parsed.line !== undefined ? ` (${text.parseFailAt.replace('{line}', String(parsed.line))})` : ''}
        </p>
      )}
      {maxRowsFallback && (
        <p className="m-0 text-xs text-[var(--color-brand-gold-500)]">
          {text.entityMaxRowsFallback.replace('{max}', String(maxRows))}
        </p>
      )}
      {existingLoadFailed && (
        <p className="m-0 text-xs text-[var(--color-brand-gold-500)]">{text.entityExistingLoadFail}</p>
      )}
      {rows && (
        <div className="mt-1 rounded-sm border border-[var(--shell-side-border)]">
          <p className="m-0 px-3 pt-2.5 text-xs text-[var(--shell-content-text)]">
            {text.previewOf.replace('{count}', String(rows.length))}
          </p>
          {dedupe.skipped > 0 && (
            <p className="m-0 px-3 pt-1 text-[11px] text-[var(--color-brand-gold-500)]">
              {text.entityDedupSkipped.replace('{count}', String(dedupe.skipped))}
            </p>
          )}
          <div className="mt-2 overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow><TableHead>#</TableHead>{def.columns.map((c) => <TableHead key={c.key}>{c.key}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {rows.slice(0, PREVIEW_ROWS).map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{i + 1}</TableCell>
                    {def.columns.map((c) => {
                      const v = r[c.key]
                      return <TableCell key={c.key}>{v === undefined ? '' : String(v)}</TableCell>
                    })}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {rows.length > PREVIEW_ROWS && (
            <p className="m-0 px-3 pb-2.5 text-[11px] text-[var(--shell-group-title)]">
              {text.previewTruncated} ({PREVIEW_ROWS})
            </p>
          )}
        </div>
      )}
      <div className="mt-3 flex flex-wrap items-center gap-3">
        <ToolbarButton primary disabled={noPerm || !rows || rows.length === 0 || busy} onClick={run}>
          {busy ? text.importing : text.importBtn}
        </ToolbarButton>
        {noPerm && <span className="text-xs text-[var(--color-danger)]">{text.entityNoPerm.replace('{perm}', def.perm)}</span>}
        {progress && !busy && <Badge variant={failures.length ? 'warning' : 'success'}>{summary}</Badge>}
        {pendingTask && !busy && (
          <ToolbarButton onClick={retryRegister}>{text.taskRetryRegister}</ToolbarButton>
        )}
        {busy && progress && (
          <span className="text-xs text-[var(--shell-group-title)]">
            {text.entityProgress.replace('{done}', String(progress.done)).replace('{total}', String(progress.total))}
          </span>
        )}
      </div>
      {error && <div className="mt-2"><ErrorBanner message={error} /></div>}
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
