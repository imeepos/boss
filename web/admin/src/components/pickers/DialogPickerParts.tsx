// DialogPicker 展示子件:筛选区 / 候选表 / 已选回显。纯展示无状态,文案与选项由 DialogPicker 注入。
import { Dropdown } from '../Dropdown'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../ui/table'
import type { DialogPickerColumn, DialogPickerFilterDef } from './DialogPicker'
import type { SelectionChip } from './pickerCore'

const FILTER_INPUT =
  'h-8 w-full max-w-[220px] rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)] disabled:cursor-not-allowed disabled:bg-[var(--shell-input-disabled-bg)] disabled:text-[var(--shell-input-placeholder)]'

const CHIP_X_BTN =
  'inline-flex h-4 w-4 shrink-0 cursor-pointer items-center justify-center rounded-full border-none bg-none p-0 text-[var(--shell-group-title)] hover:text-[var(--color-danger)]'

/** 筛选区:关键字输入 + 可配置筛选项下拉(filter options 需自带 value='' 的"全部"项)。 */
export function PickerFilterBar({ keyword, onKeyword, keywordPlaceholder, filters, values, onFilter, disabled }: {
  keyword: string
  onKeyword: (v: string) => void
  keywordPlaceholder?: string
  filters: DialogPickerFilterDef[]
  values: Record<string, string>
  onFilter: (key: string, value: string) => void
  disabled?: boolean
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 pb-3">
      <input
        type="text"
        className={FILTER_INPUT}
        aria-label={keywordPlaceholder ?? ''}
        placeholder={keywordPlaceholder}
        value={keyword}
        disabled={disabled}
        onChange={(e) => onKeyword(e.target.value)}
      />
      {filters.map((f) => (
        <Dropdown
          key={f.key}
          value={values[f.key] ?? ''}
          options={f.options}
          onChange={(v) => onFilter(f.key, v)}
          ariaLabel={f.label}
          disabled={disabled}
          triggerStyle={{ minWidth: 140 }}
        />
      ))}
    </div>
  )
}

/** 行首选择标识:单选圆点 / 多选方勾,纯视觉,选中态由行 aria-selected 与高亮表达。 */
function SelectMark({ mode, selected }: { mode: 'single' | 'multiple'; selected: boolean }) {
  if (mode === 'single') {
    return (
      <span aria-hidden className={'inline-flex h-4 w-4 items-center justify-center rounded-full border bg-none ' + (selected
        ? 'border-[var(--shell-fab-bg)]'
        : 'border-[var(--shell-input-border)]')}>
        {selected && <span className="h-2 w-2 rounded-full bg-[var(--shell-fab-bg)]" />}
      </span>
    )
  }
  return (
    <span aria-hidden className={'inline-flex h-4 w-4 items-center justify-center rounded-[3px] border ' + (selected
      ? 'border-[var(--shell-fab-bg)] bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]'
      : 'border-[var(--shell-input-border)]')}>
      {selected && (
        <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor"
          strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
          <path d="M5 12.5l4.5 4.5L19 7.5" />
        </svg>
      )}
    </span>
  )
}

/** 候选表:整行点选(点击/回车/空格),选中行高亮;列渲染缺省取同名属性字符串。 */
export function PickerTable<T>({ mode, columns, items, rowKey, selectedKeys, onRowPick }: {
  mode: 'single' | 'multiple'
  columns: DialogPickerColumn<T>[]
  items: T[]
  rowKey: (item: T) => string
  selectedKeys: string[]
  onRowPick: (item: T) => void
}) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-9" />
          {columns.map((c) => <TableHead key={c.key}>{c.title}</TableHead>)}
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => {
          const key = rowKey(item)
          const selected = selectedKeys.includes(key)
          const rowCls = 'cursor-pointer' + (selected ? ' bg-[var(--shell-menu-hover-bg)]' : '')
          return (
            <TableRow
              key={key}
              className={rowCls}
              aria-selected={selected}
              tabIndex={0}
              onClick={() => onRowPick(item)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onRowPick(item) }
              }}
            >
              <TableCell><SelectMark mode={mode} selected={selected} /></TableCell>
              {columns.map((c) => (
                <TableCell key={c.key}>{c.render ? c.render(item) : cellText(item, c.key)}</TableCell>
              ))}
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

/** 列渲染缺省:按列 key 取同名字段字符串化,空值落破折号。 */
function cellText<T>(item: T, key: string): string {
  const v = (item as Record<string, unknown>)[key]
  if (v === undefined || v === null || v === '') return '—'
  return String(v)
}

/** 已选回显:计数 + chips(单个移除)+ 清空;无选中不渲染。 */
export function PickerChips({ chips, selectedCount, removeLabel, clearAllLabel, onRemove, onClearAll }: {
  chips: SelectionChip[]
  selectedCount: string
  removeLabel: string
  clearAllLabel: string
  onRemove: (key: string) => void
  onClearAll: () => void
}) {
  if (chips.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-2 pb-1">
      <span className="text-xs text-[var(--shell-group-title)]">
        {selectedCount.replace('{n}', String(chips.length))}
      </span>
      {chips.map((c) => (
        <span
          key={c.key}
          className="inline-flex max-w-[220px] items-center gap-1 rounded-full border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] py-0.5 pl-2.5 pr-1 text-xs text-[var(--shell-content-text)]"
        >
          <span className="truncate">{c.label}</span>
          <button type="button" className={CHIP_X_BTN} aria-label={removeLabel + ': ' + c.label} onClick={() => onRemove(c.key)}>
            <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor"
              strokeWidth="2" strokeLinecap="round" aria-hidden>
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </span>
      ))}
      <button
        type="button"
        className="cursor-pointer border-none bg-none p-0 text-xs text-[var(--color-text-link)] hover:underline"
        onClick={onClearAll}
      >
        {clearAllLabel}
      </button>
    </div>
  )
}
