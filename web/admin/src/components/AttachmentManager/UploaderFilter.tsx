// 上传者筛选行:类型下拉 + 上传者 ID 输入;AttachmentManager 自由筛选模式的子视图。
import { Dropdown } from '../Dropdown'
import { INPUT } from './styles'
import { useT } from '../../i18n'

export interface UploaderFilterProps {
  typeSel: string
  onTypeChange: (type: string) => void
  uid: string
  onUidChange: (uid: string) => void
}

export function UploaderFilter({ typeSel, onTypeChange, uid, onUidChange }: UploaderFilterProps) {
  const { attachmentManager: t } = useT()
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
      <input
        className={INPUT}
        value={uid}
        inputMode="numeric"
        placeholder={t.uploaderIdPlaceholder}
        aria-label={t.uploaderIdPlaceholder}
        disabled={!typeSel}
        onChange={(e) => onUidChange(e.target.value.replace(/\D/g, ''))}
      />
    </>
  )
}
