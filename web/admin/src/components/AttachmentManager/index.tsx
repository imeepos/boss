// 附件管理通用组件:上传(多文件)/查询(关键词+上传者+文件类型+分页)/软删除/选择(复选回调)。
// 按 MIME/扩展名识别 9 类文件并渲染彩色图标徽章(image/audio/video/pdf/document/spreadsheet/archive/code/other);
// 客户端按类型过滤(后端契约不动);左侧分类侧栏 + 顶部类型下拉联动;视图切换 UI 占位(数据模型不变)。
// 嵌入场景由 props 固定上传者(隐藏筛选行);独立使用时暴露完整筛选。
// 样式:tailwind 原子类 + tokens.css .afti-* 类别色,双主题自动切换;文案走 i18n attachmentManager 块。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { LayoutGrid, List as ListIcon } from 'lucide-react'
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
import {
  FILE_CATEGORY_KEYS, classifyAttachment, countByCategory, filterByCategory,
  fmtBytes, oversizeFiles, toggleSelection, toListQuery,
  type FileCategoryKey,
} from './logic'
import { FileTypeIcon } from './FileTypeIcon'

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
  const [categorySel, setCategorySel] = useState<FileCategoryKey | ''>('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [items, setItems] = useState<AttachmentDTO[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [view, setView] = useState<'list' | 'grid'>('list')

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

  const visibleItems = useMemo(
    () => filterByCategory(items, categorySel),
    [items, categorySel],
  )
  const categoryCounts = useMemo(() => countByCategory(items), [items])
  const visibleTotal = categorySel ? visibleItems.length : total
  const pageColSpan = (selectable ? 1 : 0) + (!fixedUploader ? 1 : 0) + 5

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
    const mine = selectedIds.filter((id) => visibleItems.some((it) => it.id === id))
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

  const allChecked = selectable && visibleItems.length > 0 && visibleItems.every((it) => selectedIds.includes(it.id))
  const toggleRow = (id: number, checked: boolean) =>
    onSelectionChange?.(toggleSelection(selectedIds, id, checked))
  const toggleAll = (checked: boolean) => {
    if (!checked) {
      const pageIds = new Set(visibleItems.map((it) => it.id))
      onSelectionChange?.(selectedIds.filter((id) => !pageIds.has(id)))
      return
    }
    onSelectionChange?.([...new Set([...selectedIds, ...visibleItems.map((it) => it.id)])])
  }

  const sidebarItems = FILE_CATEGORY_KEYS
  const categoryOptions = useMemo(() => [
    { value: '', label: t.fileCategory.all },
    ...sidebarItems.map((k) => ({ value: k, label: t.fileCategory[k] })),
  ], [t, sidebarItems])

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
        <Dropdown
          value={categorySel}
          ariaLabel={t.filterByType}
          onChange={(v) => { setCategorySel(v as FileCategoryKey | ''); setPage(1) }}
          options={categoryOptions}
        />
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
            <ToolbarButton disabled>{t.downloadSelected}</ToolbarButton>
            <ToolbarButton onClick={() => void onDeleteSelected()} disabled={busy}>{t.deleteSelected}</ToolbarButton>
          </>
        )}
        <div className="inline-flex h-8 overflow-hidden rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)]">
          <button
            type="button"
            aria-label={t.colFileName + ' ' + view}
            className={`inline-flex h-full w-8 items-center justify-center transition-colors ${view === 'list' ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : 'text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'}`}
            onClick={() => setView('list')}
          >
            <ListIcon className="h-4 w-4" />
          </button>
          <button
            type="button"
            aria-label={'grid'}
            className={`inline-flex h-full w-8 items-center justify-center border-l border-[var(--shell-input-border)] transition-colors ${view === 'grid' ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : 'text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'}`}
            onClick={() => setView('grid')}
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
        </div>
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

      <div className="flex min-h-[280px]">
        {/* 左侧分类侧栏 */}
        <aside className="hidden w-56 shrink-0 border-r border-[var(--shell-side-border)] bg-[var(--shell-side-bg)] py-2 lg:block">
          <button
            type="button"
            className={`flex w-full items-center gap-2 px-4 py-2 text-left text-[13px] transition-colors ${categorySel === '' ? 'border-l-[3px] border-[var(--shell-nav-line)] bg-[var(--shell-menu-active-bg)] font-semibold text-[var(--shell-menu-active-text)]' : 'text-[var(--shell-menu-text)] hover:bg-[var(--shell-menu-hover-bg)]'}`}
            onClick={() => { setCategorySel(''); setPage(1) }}
          >
            <span className={`afti-image inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-sm ${categorySel === '' ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : ''}`}>
              <LayoutGrid className="h-3.5 w-3.5" />
            </span>
            <span className="flex-1">{t.fileCategory.all}</span>
            <span className={`min-w-6 rounded-sm px-1.5 py-0.5 text-center text-[11px] tabular-nums ${categorySel === '' ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : 'bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'}`}>{total}</span>
          </button>
          {sidebarItems.map((k) => {
            const count = categoryCounts[k]
            const active = categorySel === k
            return (
              <button
                key={k}
                type="button"
                title={`${t.fileCategory[k]} · ${t.categoryBadgeTip}`}
                className={`flex w-full items-center gap-2 px-4 py-2 text-left text-[13px] transition-colors ${active ? 'border-l-[3px] border-[var(--shell-nav-line)] bg-[var(--shell-menu-active-bg)] font-semibold text-[var(--shell-menu-active-text)]' : 'text-[var(--shell-menu-text)] hover:bg-[var(--shell-menu-hover-bg)]'}`}
                onClick={() => { setCategorySel(active ? '' : k); setPage(1) }}
              >
                <FileTypeIcon category={k} size={20} />
                <span className="flex-1">{t.fileCategory[k]}</span>
                <span className={`min-w-6 rounded-sm px-1.5 py-0.5 text-center text-[11px] tabular-nums ${active ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : 'bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'}`}>{count}</span>
              </button>
            )
          })}
        </aside>

        {/* 主区:列表 / 网格 */}
        <div className="min-w-0 flex-1">
          {view === 'list' ? (
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
                    <TableHead className="w-28">{t.colActions}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading && (
                    <TableRow><TableCell colSpan={pageColSpan} className="py-8 text-center text-[var(--shell-group-title)]">{common.loading}</TableCell></TableRow>
                  )}
                  {!loading && visibleItems.length === 0 && (
                    <TableRow><TableCell colSpan={pageColSpan} className="py-8 text-center text-[var(--shell-group-title)]">{t.empty}</TableCell></TableRow>
                  )}
                  {!loading && visibleItems.map((row) => {
                    const cat = classifyAttachment(row.contentType, row.fileName)
                    return (
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
                        <TableCell className="max-w-72 text-[var(--shell-heading)]">
                          <div className="flex items-center gap-2.5">
                            <FileTypeIcon category={cat} size={28} />
                            <span className="truncate" title={row.fileName}>{row.fileName || '—'}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-[var(--shell-content-text)]">{t.fileCategory[cat]}</TableCell>
                        <TableCell className="text-[var(--shell-content-text)] tabular-nums">{fmtBytes(row.sizeBytes)}</TableCell>
                        {!fixedUploader && (
                          <TableCell className="text-[var(--shell-content-text)]">
                            {uploaderLabel[row.uploaderType] ?? row.uploaderType} #{row.uploaderId}
                          </TableCell>
                        )}
                        <TableCell className="text-[var(--shell-content-text)]">{fmtTime(row.createdAt)}</TableCell>
                        <TableCell>
                          <div className="inline-flex items-center gap-1">
                            <button type="button" className={DANGER_BTN} title={t.download}>{t.download}</button>
                            <span className="text-[var(--shell-side-border)]">|</span>
                            <button type="button" className={DANGER_BTN} onClick={() => void onDelete(row)}>{t.delete}</button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          ) : (
            <GridView
              loading={loading}
              empty={t.empty}
              rows={visibleItems}
              selectable={selectable}
              selectedIds={selectedIds}
              fixedUploader={fixedUploader}
              uploaderLabel={uploaderLabel}
              t={t}
              onToggle={toggleRow}
              onDelete={(row) => void onDelete(row)}
            />
          )}

          {categorySel && (
            <p className="mx-4 mb-2 mt-1 text-[11px] text-[var(--shell-crumb-text)]">
              {t.filterScopeHint.replace('{filtered}', String(visibleItems.length)).replace('{total}', String(total))}
            </p>
          )}
          {err && <p className="mx-4 mb-2 mt-1 text-xs text-[var(--color-danger)]">{err}</p>}
        </div>
      </div>

      <div className="flex items-center justify-between border-t border-[var(--shell-side-border)] px-4 py-2">
        <span className="text-xs text-[var(--shell-group-title)]">
          {t.rangeText.replace('{from}', '1').replace('{to}', String(visibleTotal)).replace('{count}', String(visibleTotal))}
        </span>
        <Pagination
          page={page} pageSize={pageSize} total={visibleTotal}
          onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
          rangeText={t.rangeText} prevText={t.prev} nextText={t.next}
          perPageText={t.perPage} jumpText={t.jump} pageUnitText={t.pageUnit}
        />
      </div>
    </section>
  )
}

interface GridProps {
  loading: boolean
  empty: string
  rows: AttachmentDTO[]
  selectable: boolean
  selectedIds: number[]
  fixedUploader: boolean
  uploaderLabel: Record<UploaderType, string>
  t: ReturnType<typeof useT>['attachmentManager']
  onToggle: (id: number, checked: boolean) => void
  onDelete: (row: AttachmentDTO) => void
}

function GridView({ loading, empty, rows, selectable, selectedIds, fixedUploader, uploaderLabel, t, onToggle, onDelete }: GridProps) {
  if (loading) {
    return <div className="grid grid-cols-2 gap-3 p-4 md:grid-cols-3 xl:grid-cols-4">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="h-28 animate-pulse rounded-md bg-[var(--shell-menu-hover-bg)]" />
      ))}
    </div>
  }
  if (!rows.length) {
    return <div className="py-12 text-center text-[var(--shell-group-title)]">{empty}</div>
  }
  return (
    <div className="grid grid-cols-2 gap-3 p-4 md:grid-cols-3 xl:grid-cols-4">
      {rows.map((row) => {
        const cat = classifyAttachment(row.contentType, row.fileName)
        const selected = selectedIds.includes(row.id)
        return (
          <div
            key={row.id}
            className={`group relative flex flex-col gap-2 rounded-md border bg-[var(--shell-card-bg)] p-3 transition-colors ${selected ? 'border-[var(--shell-fab-bg)] ring-1 ring-[var(--shell-fab-bg)]' : 'border-[var(--shell-card-border)] hover:border-[var(--shell-input-border-hover)]'}`}
          >
            {selectable && (
              <input
                type="checkbox" aria-label={row.fileName}
                className="absolute right-2 top-2 accent-[var(--shell-fab-bg)]"
                checked={selected}
                onChange={(e) => onToggle(row.id, e.target.checked)}
              />
            )}
            <div className="flex items-center gap-2.5">
              <FileTypeIcon category={cat} size={36} />
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] font-medium text-[var(--shell-heading)]" title={row.fileName}>{row.fileName || '—'}</div>
                <div className="mt-0.5 flex items-center gap-2 text-[11px] text-[var(--shell-content-text)]">
                  <span className="tabular-nums">{fmtBytes(row.sizeBytes)}</span>
                  <span>·</span>
                  <span className="truncate">{fmtTime(row.createdAt)}</span>
                </div>
              </div>
            </div>
            {!fixedUploader && (
              <div className="text-[11px] text-[var(--shell-content-text)]">
                {uploaderLabel[row.uploaderType] ?? row.uploaderType} #{row.uploaderId}
              </div>
            )}
            <div className="flex items-center justify-end gap-1 border-t border-[var(--shell-side-border)] pt-2">
              <button type="button" className={DANGER_BTN} title={t.download}>{t.download}</button>
              <span className="text-[var(--shell-side-border)]">|</span>
              <button type="button" className={DANGER_BTN} onClick={() => onDelete(row)}>{t.delete}</button>
            </div>
          </div>
        )
      })}
    </div>
  )
}