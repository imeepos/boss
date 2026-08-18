// 通用分页条:总数文案 + 每页条数 + 翻页按钮。文案由调用方传入(i18n)。
import './Pagination.css'
interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onPage: (page: number) => void
  onSize: (size: number) => void
  totalText: string
  prevText: string
  nextText: string
  perPageText: string
}

const SIZE_OPTIONS = [10, 20, 50, 100]

export function Pagination({
  page, pageSize, total, onPage, onSize, totalText, prevText, nextText, perPageText,
}: PaginationProps) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const current = Math.min(page, pages)
  const from = total === 0 ? 0 : (current - 1) * pageSize + 1
  const to = Math.min(total, current * pageSize)
  return (
    <div className="pager" role="navigation" aria-label="pagination">
      <span className="pager-info">
        {totalText.replace('{count}', String(total))}
        {total > 0 && ` · ${from}-${to}`}
      </span>
      <label className="pager-size">
        <select
          value={pageSize}
          onChange={(e) => onSize(Number(e.target.value))}
          aria-label={perPageText}
        >
          {SIZE_OPTIONS.map((n) => (
            <option key={n} value={n}>{n} {perPageText}</option>
          ))}
        </select>
      </label>
      <button className="pager-btn" disabled={current <= 1} onClick={() => onPage(current - 1)}>
        {prevText}
      </button>
      <span className="pager-page">{current} / {pages}</span>
      <button className="pager-btn" disabled={current >= pages} onClick={() => onPage(current + 1)}>
        {nextText}
      </button>
    </div>
  )
}
