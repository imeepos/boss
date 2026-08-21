// 客户选择器:/customers keyword 服务端检索;详情走 GET /customers/{id};管理页 /bss/customer。
import { useT } from '../../i18n'
import { EntityPicker } from './EntityPicker'
import { getCustomerDetail, searchCustomers, type CustomerItem } from '../../api/pickers'
import { customerDetailItems } from './detailItems'

export function CustomerPicker({ value, onChange, disabled }: {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}) {
  const p = useT().pages.pickers
  return (
    <EntityPicker<CustomerItem>
      value={value}
      onChange={onChange}
      disabled={disabled}
      search={searchCustomers}
      toOption={(c) => ({ value: String(c.id), label: `${c.name} · ${c.phone || c.customerCode}` })}
      fetchDetail={() => getCustomerDetail(value)}
      detailItems={(c) => customerDetailItems(c, p.customer)}
      detailTitle={p.customer.title}
      listPath="/bss/customer"
      texts={{
        aria: p.customer.aria,
        placeholder: p.common.placeholder,
        loadFail: p.common.loadFail,
        viewDetail: p.common.viewDetail,
        detailFail: p.common.detailFail,
        close: p.common.close,
        jumpToList: p.common.jumpToList,
      }}
    />
  )
}
