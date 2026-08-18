// 通用分页条,对齐 antd Pagination 规范:
// 页码序列(首末恒显 + 当前页±2 + 省略号)、总数区间文案、每页条数、快速跳转。
// 文案由调用方传入(i18n)。单页不隐藏,禁用态置灰(确定性)。
import { useState } from 'react'
import { Dropdown } from './Dropdown'
import './Pagination.css'

interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onPage: (page: number) => void
  onSize: (size: number) => void
  /** 形如 '第 {from}-{to} 条,共 {count} 条'。 */
  rangeText: string
  prevText: string
  nextText: string
  perPageText: string
  jumpText: string
  pageUnitText: string
}

const SIZE_OPTIONS = [10, 20, 50, 100]
const QUICK_JUMP_THRESHOLD = 10

export function Pagination({
  page, pageSize, total, onPage, onSize, rangeText, prevText, nextText,
  perPageText, jumpText, pageUnitText,
}: PaginationProps) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const current = Math.min(page, pages)
  const from = total === 0 ? 0 : (current - 1) * pageSize + 1
  const to = Math.min(total, current * pageSize)
  return (
    <nav className="pager" aria-label="pagination">
      <span className="pager-total">
        {rangeText
          .replace('{from}', String(from))
          .replace('{to}', String(to))
          .replace('{count}', String(total))}
      </span>
      <div className="pager-controls">
        <button className="pager-item" disabled={current <= 1} aria-label={prevText}
          onClick={() => onPage(current - 1)}>{prevText}</button>
        {pageSequence(current, pages).map((p, i) =>
          p === '…' ? (
            <span key={`e${i}`} className="pager-ellipsis">…</span>
          ) : (
            <button
              key={p}
              className={`pager-item${p === current ? ' pager-item-active' : ''}`}
              aria-current={p === current ? 'page' : undefined}
              aria-label={`${pageUnitText} ${p}`}
              onClick={() => onPage(p)}
            >
              {p}
            </button>
          ))}
        <button className="pager-item" disabled={current >= pages} aria-label={nextText}
          onClick={() => onPage(current + 1)}>{nextText}</button>
        <SizeChanger pageSize={pageSize} onSize={onSize} perPageText={perPageText} />
        {pages > QUICK_JUMP_THRESHOLD && (
          <QuickJumper pages={pages} onPage={onPage}
            jumpText={jumpText} pageUnitText={pageUnitText} />
        )}
      </div>
    </nav>
  )
}

// pageSequence 首末页恒显,当前页 ±2,越界折叠为省略号。
function pageSequence(current: number, pages: number): (number | '…')[] {
  if (pages <= 7) return range(1, pages)
  const mid = range(Math.max(2, current - 2), Math.min(pages - 1, current + 2))
  const head: (number | '…')[] = mid[0] === 2 ? [1] : [1, '…']
  const tail: (number | '…')[] = mid[mid.length - 1] === pages - 1 ? [pages] : ['…', pages]
  return [...head, ...mid, ...tail]
}

function range(from: number, to: number): number[] {
  const out: number[] = []
  for (let i = from; i <= to; i++) out.push(i)
  return out
}

// SizeChanger 每页条数切换(antd showSizeChanger):自定义下拉,不用原生 select。
function SizeChanger({ pageSize, onSize, perPageText }: {
  pageSize: number
  onSize: (size: number) => void
  perPageText: string
}) {
  return (
    <Dropdown
      value={String(pageSize)}
      ariaLabel={perPageText}
      onChange={(v) => onSize(Number(v))}
      options={SIZE_OPTIONS.map((n) => ({ value: String(n), label: `${n} ${perPageText}` }))}
    />
  )
}

// QuickJumper 快速跳转(antd showQuickJumper):仅页数较多时出现。
function QuickJumper({ pages, onPage, jumpText, pageUnitText }: {
  pages: number
  onPage: (page: number) => void
  jumpText: string
  pageUnitText: string
}) {
  const [value, setValue] = useState('')
  const go = () => {
    const n = Number.parseInt(value, 10)
    if (Number.isFinite(n)) onPage(Math.min(pages, Math.max(1, n)))
    setValue('')
  }
  return (
    <span className="pager-jumper">
      {jumpText}
      <input value={value} aria-label={jumpText}
        onChange={(e) => setValue(e.target.value.replace(/\D/g, ''))}
        onKeyDown={(e) => e.key === 'Enter' && go()} />
      {pageUnitText}
      <button className="pager-item" onClick={go}>Go</button>
    </span>
  )
}
