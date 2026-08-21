// 用户选择器:/users keyword 服务端检索;详情走 GET /users/{customerId} 聚合摘要;管理页 /bss/user。
import { useT } from '../../i18n'
import { EntityPicker } from './EntityPicker'
import { getUserDetail, searchUsers, type UserItem } from '../../api/pickers'
import { userDetailItems } from './detailItems'

export function UserPicker({ value, onChange, disabled }: {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}) {
  const p = useT().pages.pickers
  return (
    <EntityPicker<UserItem>
      value={value}
      onChange={onChange}
      disabled={disabled}
      search={searchUsers}
      toOption={(u) => ({ value: String(u.customerId), label: `${u.name} · ${u.phone}` })}
      fetchDetail={() => getUserDetail(value)}
      detailItems={(u) => userDetailItems(u, p.user)}
      detailTitle={p.user.title}
      listPath="/bss/user"
      texts={{
        aria: p.user.aria,
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
