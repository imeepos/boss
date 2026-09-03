// 大数据量弹框选择器基座:服务端分页表格 + 多维搜索(关键字 + 可配置筛选项)+ 单/多选。
// 受控 open/onPick/onClose;列定义与查询函数由调用方注入,组件不绑定业务域;
// 布局对齐附件选择器弹框(筛选区 + 列表区 + 底部确认);文案(含 aria)一律由调用方 i18n 传入。
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '../ui/dialog'
import { Pagination } from '../Pagination'
import { Loading } from '../Loading'
import { ToolbarButton } from '../business/page-head'
import { togglePickKey, pickSingleKey, toSelectionChips } from './pickerCore'
import { PickerChips, PickerFilterBar, PickerTable } from './DialogPickerParts'

export interface DialogPickerColumn<T> {
  /** 列标识(缺省渲染时取实体同名属性)。 */
  key: string
  /** 列头文案(调用方 i18n)。 */
  title: string
  /** 自定义单元格渲染;缺省按 key 取属性字符串化。 */
  render?: (item: T) => ReactNode
}

export interface DialogPickerFilterDef {
  /** 筛选参数名,随查询参数上抛。 */
  key: string
  /** 筛选项标签(调用方 i18n),兼作下拉 aria-label。 */
  label: string
  /** 候选值;需自带 value='' 的"全部"项作为首项。 */
  options: Array<{ value: string; label: string }>
}

/** 服务端分页查询入参:关键字 + 筛选项 + 页码。 */
export interface DialogPickerQuery {
  keyword: string
  filters: Record<string, string>
  page: number
  pageSize: number
}

/** 服务端分页查询出参:当页数据 + 总数。 */
export interface DialogPickerPage<T> {
  items: T[]
  total: number
}

export interface DialogPickerTexts {
  confirm: string
  cancel: string
  loadFail: string
  empty: string
  /** 关键字输入 placeholder 与 aria。 */
  keywordPh: string
  /** 已选计数模板,含 {n}。 */
  selectedCount: string
  clearAll: string
  /** 单个移除按钮 aria 前缀。 */
  remove: string
  /** 分页条文案,对齐 Pagination 组件契约。 */
  pager: {
    rangeText: string; prev: string; next: string
    perPage: string; jump: string; pageUnit: string
  }
}

export interface DialogPickerProps<T> {
  open: boolean
  /** 默认单选;multiple 支持跨页累选。 */
  mode?: 'single' | 'multiple'
  /** 弹框标题(调用方 i18n)。 */
  title: string
  onClose: () => void
  /** 确认回传:单选长度 1,多选按选择顺序。 */
  onPick: (items: T[]) => void
  columns: DialogPickerColumn<T>[]
  /** 服务端分页查询函数(调用方注入域检索)。 */
  query: (q: DialogPickerQuery) => Promise<DialogPickerPage<T> | null>
  rowKey: (item: T) => string
  /** 已选回显文案,如"张三 · 138xxxx"。 */
  rowLabel: (item: T) => string
  /** 附加筛选项(与关键字构成多维搜索)。 */
  filters?: DialogPickerFilterDef[]
  initialPageSize?: number
  texts: DialogPickerTexts
}

const KEYWORD_DEBOUNCE_MS = 300
const DEFAULT_PAGE_SIZE = 10

/** 列表数据流:关键字防抖 / 条件变页重置 / 服务端分页查询(seq 防竞态)。 */
function usePickerDialogData<T>(args: {
  open: boolean
  query: (q: DialogPickerQuery) => Promise<DialogPickerPage<T> | null>
  initialPageSize?: number
}) {
  const { open, query, initialPageSize } = args
  const [keyword, setKeyword] = useState('')
  const [committed, setCommitted] = useState('')
  const [filterVals, setFilterVals] = useState<Record<string, string>>({})
  const [page, setPageRaw] = useState(1)
  const [pageSize, setPageSizeRaw] = useState(initialPageSize ?? DEFAULT_PAGE_SIZE)
  const [items, setItems] = useState<T[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [loadErr, setLoadErr] = useState(false)
  const seqRef = useRef(0)
  const queryRef = useRef(query)
  queryRef.current = query

  useEffect(() => {
    if (!open) return
    setKeyword(''); setCommitted(''); setFilterVals({}); setPageRaw(1)
    setItems([]); setTotal(0); setLoading(false); setLoadErr(false)
  }, [open])

  useEffect(() => {
    const timer = setTimeout(() => setCommitted(keyword.trim()), KEYWORD_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [keyword])

  useEffect(() => { setPageRaw(1) }, [committed, filterVals])

  useEffect(() => {
    if (!open) return undefined
    const seq = ++seqRef.current
    setLoading(true)
    queryRef.current({ keyword: committed, filters: filterVals, page, pageSize })
      .then((res) => {
        if (seq !== seqRef.current) return
        setItems(res?.items ?? []); setTotal(res?.total ?? 0); setLoading(false)
      })
      .catch(() => {
        if (seq !== seqRef.current) return
        setItems([]); setTotal(0); setLoadErr(true); setLoading(false)
      })
    return () => { seqRef.current += 1 }
  }, [open, committed, filterVals, page, pageSize])

  const onFilter = (key: string, value: string) => setFilterVals((prev) => ({ ...prev, [key]: value }))
  const onPage = (p: number) => setPageRaw(p)
  const onPageSize = (n: number) => { setPageSizeRaw(n); setPageRaw(1) }
  return { keyword, onKeyword: setKeyword, filterVals, onFilter, page, pageSize, onPage, onPageSize, items, total, loading, loadErr }
}

/** 选择集合:单选替换 / 多选累选(跨页保持),key→label/item 索引增量维护供回显与回传。 */
function usePickerSelection<T>(mode: 'single' | 'multiple', rowKey: (item: T) => string, rowLabel: (item: T) => string) {
  const [selected, setSelected] = useState<string[]>([])
  const keyToLabel = useRef(new Map<string, string>())
  const keyToItem = useRef(new Map<string, T>())

  const onRowPick = (item: T) => {
    const key = rowKey(item)
    setSelected((prev) => (mode === 'multiple'
      ? togglePickKey(prev, key, !prev.includes(key))
      : pickSingleKey(key)))
    keyToLabel.current.set(key, rowLabel(item))
    keyToItem.current.set(key, item)
  }
  const onRemoveChip = (key: string) => setSelected((prev) => togglePickKey(prev, key, false))
  const onClearAll = () => setSelected([])
  const reset = () => {
    setSelected([])
    keyToLabel.current.clear()
    keyToItem.current.clear()
  }
  const chips = useMemo(() => toSelectionChips(selected, keyToLabel.current), [selected])
  const pickedItems = useMemo(
    () => selected.map((k) => keyToItem.current.get(k)).filter((x): x is T => x !== undefined),
    [selected],
  )
  return { selected, onRowPick, onRemoveChip, onClearAll, reset, chips, pickedItems }
}

export function DialogPicker<T>({
  open, mode = 'single', title, onClose, onPick, columns, query,
  rowKey, rowLabel, filters = [], initialPageSize, texts,
}: DialogPickerProps<T>) {
  const data = usePickerDialogData<T>({ open, query, initialPageSize })
  const sel = usePickerSelection<T>(mode, rowKey, rowLabel)
  useEffect(() => {
    if (open) sel.reset()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])
  const confirm = () => {
    onPick(sel.pickedItems)
    sel.reset()
    onClose()
  }
  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-[min(64rem,calc(100vw-32px))] max-h-[90vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0">
        <DialogHeader className="px-6 pt-5 pb-3">
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div className="min-h-0 overflow-auto px-6 pb-3">
          <PickerFilterBar
            keyword={data.keyword}
            onKeyword={data.onKeyword}
            keywordPlaceholder={texts.keywordPh}
            filters={filters}
            values={data.filterVals}
            onFilter={data.onFilter}
          />
          {data.loadErr && <div className="py-2 text-xs text-[var(--color-danger)]" role="alert">{texts.loadFail}</div>}
          {data.loading ? (
            <Loading />
          ) : !data.loadErr && data.items.length === 0 ? (
            <div className="py-10 text-center text-[13px] text-[var(--shell-group-title)]">{texts.empty}</div>
          ) : (
            <PickerTable
              mode={mode}
              columns={columns}
              items={data.items}
              rowKey={rowKey}
              selectedKeys={sel.selected}
              onRowPick={sel.onRowPick}
            />
          )}
          <Pagination
            page={data.page}
            pageSize={data.pageSize}
            total={data.total}
            onPage={data.onPage}
            onSize={data.onPageSize}
            rangeText={texts.pager.rangeText}
            prevText={texts.pager.prev}
            nextText={texts.pager.next}
            perPageText={texts.pager.perPage}
            jumpText={texts.pager.jump}
            pageUnitText={texts.pager.pageUnit}
          />
        </div>
        <DialogFooter className="flex-col items-stretch gap-3 border-t border-[var(--shell-side-border)] px-6 py-3 sm:flex-col">
          <PickerChips
            chips={sel.chips}
            selectedCount={texts.selectedCount}
            removeLabel={texts.remove}
            clearAllLabel={texts.clearAll}
            onRemove={sel.onRemoveChip}
            onClearAll={sel.onClearAll}
          />
          <div className="flex items-center justify-end gap-3">
            <ToolbarButton primary disabled={sel.selected.length === 0} onClick={confirm}>
              {texts.confirm}
            </ToolbarButton>
            <ToolbarButton onClick={onClose}>{texts.cancel}</ToolbarButton>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
