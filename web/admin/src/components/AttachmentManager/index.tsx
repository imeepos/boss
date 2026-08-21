// 附件管理通用组件:上传(多文件)/查询(关键词+上传者+分页)/软删除/选择(复选回调)。
// 嵌入场景由 props 固定上传者(隐藏筛选行);独立使用时暴露完整筛选。
// 样式:tailwind 原子类,令牌走 shell-*/shadcn 双主题体系;文案走 i18n attachmentManager 块。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useT } from '../../i18n'
import { ApiError } from '../../api/envelope'
import {
  deleteAttachment, listAttachments, uploadAttachment, type AttachmentDTO, type UploaderType,
} from '../../api/attachments'
import { useConfirm } from '../ConfirmDialog'
import { Dropdown } from '../Dropdown'
import { Pagination } from '../Pagination'
import { ToolbarButton } from '../business/page-head'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../ui/table'
import { fmtTime } from '../../lib/format'
import { fmtBytes, oversizeFiles, toggleSelection, toListQuery } from './logic'

export interface AttachmentManagerProps {
  /** 固定上传者筛选(类型+id 成对传入即锁定);缺省自由筛选。 */
  uploaderType?: UploaderType
  uploaderId?: number
  /** 选择模式:行首复选 + 选中集回调。 */
  selectable?: boolean
  selectedIds?: number[]
  onSelectionChange?: (ids: number[]) => void
}

const CARD = 'border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const INPUT = 'h-8 w-44 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] placeholder:text-[var(--shell-crumb-text)] focus:border-[var(--shell-input-border-focus)] focus:outline-none'
const DANGER_BTN = 'h-7 cursor-pointer rounded-sm border-0 bg-transparent px-2 text-xs text-[var(--color-danger)] hover:underline disabled:cursor-not-allowed disabled:opacity-50'
const ERR_MSG = 'text-xs text-[var(--color-danger)]'

export function AttachmentManager({
  uploaderType, uploaderId, selectable = false, selectedIds = [], onSelectionChange,
}: AttachmentManagerProps) {
  const { attachmentManager: t, common } = useT()
  const confirmDialog = useConfirm()
  const fileRef = useRef<HTMLInputElement>(null)

  const [kwInput, setKwInput] = useState('')
  const [kw, setKw] = useState('')
  const [typeSel, setTypeSel] = useState('')
  const [uidInput, setUidInput] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [items, setItems] = useState<AttachmentDTO[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const fixedUploader = !!(uploaderType && uploaderId && uploaderId > 0)
  const uidNum = Number(uidInput)
  const effectiveType = fixedUploader ? uploaderType! : typeSel
  const effectiveId = fixedUploader ? uploaderId! : typeSel && Number.isInteger(uidNum) && uidNum > 0 ? uidNum : null

  const load = useCallback(async (targetPage: number) => {
    setLoading(true)
    setErr('')
    try {
      const data = await listAttachments(toListQuery({
        uploaderType: effectiveType, uploaderId: effectiveId,
        keyword: kw, page: targetPage, pageSize,
      }))
      setItems(data.items ?? [])
      setTotal(data.total ?? 0)
    } catch (e) {
      setItems([])
      setTotal(0)
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [effectiveType, effectiveId, kw, pageSize])

  useEffect(() => { void load(page) }, [load, page])

  /** 动作后刷新:首页直接重载,否则翻回 1 由 effect 触发。 */
  const refresh = useCallback(() => {
    if (page === 1) void load(1)
    else setPage(1)
  }, [page, load])

  const uploaderLabel = useMemo(() => ({
    account: t.uploaderAccount, worker: t.uploaderWorker, customer: t.uploaderCustomer,
  }), [t])

  const onPickFiles = async (files: File[]) => {
    if (!files.length) return
    const oversize = oversizeFiles(files)
    if (oversize.length) {
      setErr(t.uploadFail.replace('{names}', oversize.join(', ')))
      return
    }
    setBusy(true)
    setErr('')
    const failed: string[] = []
    for (const f of files) {
      try { await uploadAttachment(f) } catch { failed.push(f.name) }
    }
    setBusy(false)
    if (failed.length) setErr(t.uploadFail.replace('{names}', failed.join(', ')))
    refresh()
  }

  const onDelete = async (row: AttachmentDTO) => {
    if (!(await confirmDialog(t.confirmDeleteOne.replace('{name}', row.fileName), { danger: true }))) return
    try {
      await deleteAttachment(row.id)
      refresh()
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t.deleteFail)
    }
  }

  const onDeleteSelected = async () => {
    const mine = selectedIds.filter((id) => items.some((it) => it.id === id))
    if (!mine.length) return
    if (!(await confirmDialog(t.confirmDeleteSelected.replace('{count}', String(mine.length)), { danger: true }))) return
    setBusy(true)
    for (const id of mine) {
      try { await deleteAttachment(id) } catch { /* 逐个尽力删,失败项以列表为准 */ }
    }
    setBusy(false)
    onSelectionChange?.([])
    refresh()
  }

  const allChecked = selectable && items.length > 0 && items.every((it) => selectedIds.includes(it.id))
  const toggleRow = (id: number, checked: boolean) =>
    onSelectionChange?.(toggleSelection(selectedIds, id, checked))
  const toggleAll = (checked: boolean) => {
    if (!checked) {
      const pageIds = new Set(items.map((it) => it.id))
      onSelectionChange?.(selectedIds.filter((id) => !pageIds.has(id)))
      return
    }
    onSelectionChange?.([...new Set([...selectedIds, ...items.map((it) => it.id)])])
  }
  const colSpan = (selectable ? 1 : 0) + (!fixedUploader ? 1 : 0) + 5

  return (
    <section className={CARD} aria-label={t.title}>
      <header className="flex flex-wrap items-center gap-2.5 border-b border-[var(--shell-side-border)] px-4 py-3">
        <h3 className="m-0 text-sm font-semibold text-[var(--shell-heading)]">{t.title}</h3>
        <input
          className={INPUT}
          value={kwInput}
          placeholder={t.searchPlaceholder}
          aria-label={t.searchPlaceholder}
          onChange={(e) => setKwInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { setKw(kwInput.trim()); setPage(1) } }}
        />
        <ToolbarButton onClick={() => { setKw(kwInput.trim()); setPage(1) }}>{t.search}</ToolbarButton>
        {!fixedUploader && (
          <>
            <Dropdown
              value={typeSel}
              ariaLabel={t.allUploaders}
              onChange={(v) => { setTypeSel(v); setPage(1) }}
              options={[
                { value: '', label: t.allUploaders },
                { value: 'account', label: t.uploaderAccount },
                { value: 'worker', label: t.uploaderWorker },
                { value: 'customer', label: t.uploaderCustomer },
              ]}
            />
            <input
              className={INPUT}
              value={uidInput}
              inputMode="numeric"
              placeholder={t.uploaderIdPlaceholder}
              aria-label={t.uploaderIdPlaceholder}
              disabled={!typeSel}
              onChange={(e) => setUidInput(e.target.value.replace(/\D/g, ''))}
            />
          </>
        )}
        <span className="flex-1" />
        {selectable && selectedIds.length > 0 && (
          <>
            <span className="text-xs text-[var(--shell-group-title)]">
              {t.selectedInfo.replace('{count}', String(selectedIds.length))}
            </span>
            <ToolbarButton onClick={() => void onDeleteSelected()} disabled={busy}>{t.deleteSelected}</ToolbarButton>
          </>
        )}
        <ToolbarButton onClick={() => fileRef.current?.click()} disabled={busy}>
          {busy ? t.uploading : t.upload}
        </ToolbarButton>
        <input
          ref={fileRef} type="file" multiple className="hidden"
          aria-label={t.upload}
          onClick={(e) => { e.currentTarget.value = '' }}
          onChange={(e) => { void onPickFiles(Array.from(e.target.files ?? [])) }}
        />
      </header>

      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              {selectable && (
                <TableHead className="w-9">
                  <input type="checkbox" aria-label={t.title} className="accent-[var(--shell-fab-bg)]" checked={allChecked} onChange={(e) => toggleAll(e.target.checked)} />
                </TableHead>
              )}
              <TableHead>{t.colFileName}</TableHead>
              <TableHead>{t.colType}</TableHead>
              <TableHead>{t.colSize}</TableHead>
              {!fixedUploader && <TableHead>{t.colUploader}</TableHead>}
              <TableHead>{t.colTime}</TableHead>
              <TableHead className="w-20">{t.colActions}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && (
              <TableRow><TableCell colSpan={colSpan} className="py-8 text-center text-[var(--shell-group-title)]">{common.loading}</TableCell></TableRow>
            )}
            {!loading && items.length === 0 && (
              <TableRow><TableCell colSpan={colSpan} className="py-8 text-center text-[var(--shell-group-title)]">{t.empty}</TableCell></TableRow>
            )}
            {!loading && items.map((row) => (
              <TableRow key={row.id}>
                {selectable && (
                  <TableCell>
                    <input
                      type="checkbox" aria-label={row.fileName}
                      className="accent-[var(--shell-fab-bg)]"
                      checked={selectedIds.includes(row.id)}
                      onChange={(e) => toggleRow(row.id, e.target.checked)}
                    />
                  </TableCell>
                )}
                <TableCell className="max-w-72 truncate text-[var(--shell-heading)]" title={row.fileName}>{row.fileName || '—'}</TableCell>
                <TableCell className="text-[var(--shell-content-text)]">{row.contentType || '—'}</TableCell>
                <TableCell className="text-[var(--shell-content-text)]">{fmtBytes(row.sizeBytes)}</TableCell>
                {!fixedUploader && (
                  <TableCell className="text-[var(--shell-content-text)]">
                    {uploaderLabel[row.uploaderType] ?? row.uploaderType} #{row.uploaderId}
                  </TableCell>
                )}
                <TableCell className="text-[var(--shell-content-text)]">{fmtTime(row.createdAt)}</TableCell>
                <TableCell>
                  <button type="button" className={DANGER_BTN} onClick={() => void onDelete(row)}>{t.delete}</button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {err && <p className={`m-0 px-4 py-2 ${ERR_MSG}`}>{err}</p>}

      <div className="border-t border-[var(--shell-side-border)] px-4 py-2">
        <Pagination
          page={page} pageSize={pageSize} total={total}
          onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
          rangeText={t.rangeText} prevText={t.prev} nextText={t.next}
          perPageText={t.perPage} jumpText={t.jump} pageUnitText={t.pageUnit}
        />
      </div>
    </section>
  )
}
