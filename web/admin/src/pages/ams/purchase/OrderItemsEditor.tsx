// 采购单明细行编辑器(创建/编辑抽屉共用):物料/规格/数量/单价四列 + 添加明细。
import { useT } from '../../../i18n'
import type { OrderItemRow } from '../types'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function OrderItemsEditor({ items, onChange, onDelete }: {
  items: OrderItemRow[]
  onChange: (items: OrderItemRow[]) => void
  onDelete?: (idx: number) => void
}) {
  const d = useT().pages.purchasePage
  const set = (idx: number, patch: Partial<OrderItemRow>) =>
    onChange(items.map((x, i) => (i === idx ? { ...x, ...patch } : x)))
  const del = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--color-danger)] hover:border-[var(--color-border-hover)] disabled:cursor-not-allowed disabled:opacity-50'
  return (
    <div>
      <div className="mb-2 text-[13px] text-[var(--shell-content-text)]">{d.items}</div>
      {items.map((it, idx) => (
        <div key={idx} className="mb-2 grid grid-cols-12 gap-2">
          <input placeholder={d.phMaterial} value={it.materialCode}
            onChange={(e) => set(idx, { materialCode: e.target.value })}
            className={input + ' col-span-4'} />
          <input placeholder={d.spec} value={it.spec}
            onChange={(e) => set(idx, { spec: e.target.value })}
            className={input + ' col-span-3'} />
          <input type="number" placeholder={d.qty} value={it.quantity}
            onChange={(e) => set(idx, { quantity: Number(e.target.value) })}
            className={input + ' col-span-2'} />
          <input type="number" placeholder={d.unitAmount} value={it.unitAmount}
            onChange={(e) => set(idx, { unitAmount: Number(e.target.value) })}
            className={input + ' col-span-2'} />
          {onDelete && (
            <button type="button" className={del + ' col-span-1'} disabled={items.length <= 1}
              onClick={() => onDelete(idx)}>{d.delRow}</button>
          )}
        </div>
      ))}
      <button
        type="button"
        className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
        onClick={() => onChange([...items, { materialCode: '', spec: '', quantity: 1, unitAmount: 0 }])}
      >
        {d.addItem}
      </button>
    </div>
  )
}
