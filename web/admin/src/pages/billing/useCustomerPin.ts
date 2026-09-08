// 客户筛选钉选回显 hook(W0 选择器基线交接项):页面仅持有 customerId,触发器
// 回显需要 label——经 GET /customers/{id} 取「姓名 · 手机号」构 pinnedOptions;
// 名称未到达/接口失败时降级 #id 留痕(后端债务:列表接口无名称快照,不硬造)。
import { useEffect, useState } from 'react'
import type { DropdownOption } from '../../components/Dropdown'
import { getCustomerDetail } from '../../api/pickers'

export function useCustomerPin(customerId: string): DropdownOption[] {
  const [label, setLabel] = useState('')
  useEffect(() => {
    if (!customerId) { setLabel(''); return }
    let alive = true
    getCustomerDetail(customerId)
      .then((c) => { if (alive && c) setLabel(`${c.name} · ${c.phone || c.customerCode}`) })
      .catch(() => {})
    return () => { alive = false }
  }, [customerId])
  return customerId ? [{ value: customerId, label: label || `#${customerId}` }] : []
}
