// 上传者筛选行:类型下拉 + 类型联动选择器。
// 师傅/客户走 api/pickers 服务端关键字检索;账号列表接口(/accounts)无 keyword 参数,
// 懒加载一次后本地过滤。类型互切清空已选(不同主体域 ID 不通用),由调用方在 onTypeChange 处理。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../api/client'
import { searchCustomers, searchWorkers } from '../../api/pickers'
import { Dropdown, type DropdownOption } from '../Dropdown'
import { SimplePicker } from '../pickers/SimplePicker'
import { useT } from '../../i18n'

/** /accounts 行的最小投影(选择器只用 id/账号名/姓名)。 */
interface AccountOption {
  id: number
  username: string
  realName: string
}

export interface UploaderFilterProps {
  typeSel: string
  onTypeChange: (type: string) => void
  uid: string
  onUidChange: (uid: string) => void
}

export function UploaderFilter({ typeSel, onTypeChange, uid, onUidChange }: UploaderFilterProps) {
  const { attachmentManager: t, pages } = useT()
  const clearLabel = pages.pickers.common.clear
  const [accounts, setAccounts] = useState<AccountOption[] | null>(null)
  const [acctError, setAcctError] = useState(false)

  // 账号数据源懒加载:类型切到 account 且无缓存时拉一次;失败置错误态可重试。
  useEffect(() => {
    if (typeSel !== 'account' || accounts !== null || acctError) return
    let alive = true
    apiFetch<AccountOption[]>('/accounts')
      .then((d) => { if (alive) setAccounts(Array.isArray(d) ? d : []) })
      .catch(() => { if (alive) setAcctError(true) })
    return () => { alive = false }
  }, [typeSel, accounts, acctError])
  const retryAccounts = () => { setAcctError(false); setAccounts(null) }

  const accountOptions = useMemo<DropdownOption[]>(
    () => (accounts ?? []).map((a) => ({ value: String(a.id), label: a.username + '(' + a.realName + ')' })),
    [accounts],
  )
  const workerSearch = async (keyword: string): Promise<DropdownOption[] | null> =>
    ((await searchWorkers(keyword)) ?? []).map((w) => ({ value: String(w.id), label: w.name + ' (' + w.staffNo + ')' }))
  const customerSearch = async (keyword: string): Promise<DropdownOption[] | null> =>
    ((await searchCustomers(keyword)) ?? []).map((c) => ({ value: String(c.id), label: c.name + ' (' + c.customerCode + ')' }))

  const idAria = t.uploaderIdPlaceholder
  return (
    <>
      <Dropdown
        value={typeSel}
        ariaLabel={t.allUploaders}
        onChange={onTypeChange}
        options={[
          { value: '', label: t.allUploaders },
          { value: 'account', label: t.uploaderAccount },
          { value: 'worker', label: t.uploaderWorker },
          { value: 'customer', label: t.uploaderCustomer },
        ]}
      />
      <SimplePicker
        key={typeSel || 'none'}
        value={uid}
        onChange={onUidChange}
        ariaLabel={idAria}
        placeholder={idAria}
        clearable
        clearLabel={clearLabel}
        minWidth={200}
        {...(typeSel === 'account'
          ? {
              options: accountOptions,
              loading: accounts === null && !acctError,
              error: acctError,
              onRetry: retryAccounts,
            }
          : {
              search: typeSel === 'worker' ? workerSearch : typeSel === 'customer' ? customerSearch : undefined,
              disabled: !typeSel,
              searchPlaceholder: idAria,
            })}
      />
    </>
  )
}
