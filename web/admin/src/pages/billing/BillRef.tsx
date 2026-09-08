// 账单人读引用(billing 域共享):billId → 账单号+客户名(billMap 命中);
// 充值/预存(billId=0)显示 noBillText,未命中(列表截断)回退 #id 留痕。
import type { BillRow } from './types'
import { IdRef } from '../../components/business'

export function BillRef({ billId, map, noBillText }: {
  billId: number
  map: Map<number, BillRow>
  noBillText: string
}) {
  if (!billId) return <span className="text-[var(--shell-group-title)]">{noBillText}</span>
  const b = map.get(billId)
  if (!b) return <IdRef value={billId} />
  return (
    <span>
      {b.billNo}
      <span className="ml-1 text-[var(--shell-group-title)]">{b.customerName}</span>
    </span>
  )
}
