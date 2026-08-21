// 附件管理通用组件:上传(多文件)/查询(关键词+上传者+文件类型+分页)/软删除/选择(复选回调)。
// 本文件只承担状态与编排:头部工具栏 + 侧栏/列表/网格三个子视图;
// 分类算法在 logic.ts,图标在 FileTypeIcon.tsx,纯视图在 CategorySidebar/ListView/GridView。
// 嵌入场景由 props 固定上传者(隐藏筛选行);独立使用时暴露完整筛选。
// 样式:tailwind 原子类 + tokens.css .afti-* 类别色,双主题自动切换;文案走 i18n attachmentManager 块。
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
import {
  FILE_CATEGORY_KEYS, filterByCategory, oversizeFiles, toggleSelection, toListQuery,
  type FileCategoryKey,
} from './logic'
import { CategorySidebar } from './CategorySidebar'
import { ListView } from './ListView'
import { GridView } from './GridView'
import { ViewToggle } from './ViewToggle'
import { INPUT, CARD } from './styles'

export interface AttachmentManagerProps {
  /** 固定上传者筛选(类型+id 成对传入即锁定);缺省自由筛选。 */
  uploaderType?: UploaderType
  uploaderId?: number
  /** 选择模式:行首复选 + 选中集回调。 */
  selectable?: boolean
  selectedIds?: number[]
  onSelectionChange?: (ids: number[]) => void
}

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
  const selectCategory = (c: FileCategoryKey | '') => {
    setCategorySel(c)
    setPage(1)
  }

  const categoryOptions = useMemo(() => [
    { value: '', label: t.fileCategory.all },
    ...FILE_CATEGORY_KEYS.map((k) => ({ value: k, label: t.fileCategory[k] })),
  ], [t])

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
          onChange={(v) => selectCategory(v as FileCategoryKey | '')}
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
        <ViewToggle view={view} onChange={setView} listLabel={t.colFileName} />
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
        <CategorySidebar
          items={items}
          total={total}
          selected={categorySel}
          onSelect={selectCategory}
          t={t}
        />

        {/* 主区:列表 / 网格 */}
        <div className="min-w-0 flex-1">
          {view === 'list' ? (
            <ListView
              loading={loading}
              rows={visibleItems}
              selectable={selectable}
              selectedIds={selectedIds}
              fixedUploader={fixedUploader}
              colSpan={pageColSpan}
              allChecked={allChecked}
              uploaderLabel={uploaderLabel}
              t={t}
              commonLoading={common.loading}
              onToggle={toggleRow}
              onToggleAll={toggleAll}
              onDelete={(row) => void onDelete(row)}
            />
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
