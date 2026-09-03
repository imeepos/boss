// 级联选择器列组件：ColumnShell（带搜索框列容器）+ 国家列 / 区划列 / 直搜结果。
// 纯展示，状态由 RegionCascadePicker 上抛；样式走 shell 令牌，双主题自适应。
import type { ReactNode } from 'react'
import { Input } from '../ui/input'
import type { CountryLite, SubdivRow } from './regionCascadeCore'

export const ROW_CLS = 'flex w-full items-center justify-between gap-2 rounded-sm border-none bg-none px-2 py-1 text-left text-[13px] cursor-pointer hover:bg-[var(--shell-menu-hover-bg)]'
export const ROW_ACTIVE = ' font-semibold text-[var(--shell-fab-bg)]'

export function ColumnShell({ label, kw, onKw, placeholder, children }: {
  label: string; kw: string; onKw: (v: string) => void; placeholder: string; children: ReactNode
}) {
  return (
    <div className="flex min-w-40 max-w-56 flex-1 flex-col gap-1 rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-2">
      <p className="truncate text-[12px] font-semibold text-[var(--shell-heading)]">{label}</p>
      <Input className="h-7" placeholder={placeholder} value={kw} onChange={(e) => onKw(e.target.value)} />
      <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto">{children}</div>
    </div>
  )
}

export function CountryColumn({ countries, value, kw, onKw, onSelect, label, placeholder, disabled }: {
  countries: CountryLite[]; value: string; kw: string; onKw: (v: string) => void
  onSelect: (code: string) => void; label: string; placeholder: string; disabled?: boolean
}) {
  const k = kw.trim().toLowerCase()
  const visible = k ? countries.filter((c) => `${c.alpha2} ${c.displayName}`.toLowerCase().includes(k)) : countries
  return (
    <ColumnShell label={label} kw={kw} onKw={onKw} placeholder={placeholder}>
      {visible.map((c) => (
        <button key={c.alpha2} type="button" disabled={disabled}
          className={ROW_CLS + (c.alpha2 === value ? ROW_ACTIVE : '')} onClick={() => onSelect(c.alpha2)}>
          <span className="truncate">{c.alpha2} {c.displayName}</span>
        </button>
      ))}
    </ColumnShell>
  )
}

export function LevelColumn({ label, rows, loading, kw, onKw, onSelect, selected, placeholder, emptyText, loadingText, disabled }: {
  label: string; rows: SubdivRow[]; loading: boolean; kw: string; onKw: (v: string) => void
  onSelect: (n: SubdivRow) => void; selected: string; placeholder: string; emptyText: string
  loadingText: string; disabled?: boolean
}) {
  const k = kw.trim().toLowerCase()
  const visible = k ? rows.filter((r) => `${r.code} ${r.name}`.toLowerCase().includes(k)) : rows
  return (
    <ColumnShell label={label} kw={kw} onKw={onKw} placeholder={placeholder}>
      {loading
        ? <p className="text-xs text-[var(--shell-group-title)]">{loadingText}</p>
        : visible.length === 0
          ? <p className="text-xs text-[var(--shell-group-title)]">{emptyText}</p>
          : visible.map((r) => (
            <button key={r.code} type="button" disabled={disabled}
              className={ROW_CLS + (r.code === selected ? ROW_ACTIVE : '')} onClick={() => onSelect(r)}>
              <span className="truncate">{r.name}</span>
              <span className="flex-none text-[11px] text-[var(--shell-group-title)]">{r.code}</span>
            </button>
          ))}
    </ColumnShell>
  )
}

export function DirectResults({ rows, busy, loadingText, emptyText, onPick, disabled }: {
  rows: SubdivRow[]; busy: boolean; loadingText: string; emptyText: string
  onPick: (n: SubdivRow) => void; disabled?: boolean
}) {
  return (
    <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-2">
      {busy
        ? <p className="text-xs text-[var(--shell-group-title)]">{loadingText}</p>
        : rows.length === 0
          ? <p className="text-xs text-[var(--shell-group-title)]">{emptyText}</p>
          : rows.map((r) => (
            <button key={r.code} type="button" disabled={disabled}
              className={ROW_CLS} onClick={() => onPick(r)}>
              <span className="truncate">{r.name}</span>
              <span className="flex-none text-[11px] text-[var(--shell-group-title)]">L{r.level} {r.code}</span>
            </button>
          ))}
    </div>
  )
}
