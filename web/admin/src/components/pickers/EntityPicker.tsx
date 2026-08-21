// 实体选择器基座:服务端检索下拉 + 选中后"详情/前往管理页"入口。
// 详情走各域单档接口(抽屉展示);跳转用 react-router 导航到对应管理页。
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ResourcePicker } from '../ResourcePicker'
import { DetailDrawer } from '../business/detail-drawer'
import type { DropdownOption } from '../Dropdown'
import type { DetailItem } from '../../pages/org/shared'

export interface EntityPickerTexts {
  aria: string
  placeholder: string
  loadFail: string
  viewDetail: string
  detailFail: string
  close: string
  jumpToList: string
}

export interface EntityPickerProps<T> {
  value: string
  onChange: (value: string) => void
  search: (keyword: string) => Promise<T[] | null>
  toOption: (item: T) => DropdownOption
  fetchDetail: () => Promise<T | null>
  detailItems: (detail: T) => DetailItem[]
  detailTitle: string
  listPath: string
  texts: EntityPickerTexts
  emptyLabel?: string
  disabled?: boolean
  minWidth?: number
}

const LINK_BTN = 'h-8 cursor-pointer rounded-sm border border-none bg-none px-2 text-xs whitespace-nowrap text-primary hover:underline disabled:cursor-not-allowed disabled:opacity-50'

export function EntityPicker<T>(props: EntityPickerProps<T>) {
  const { value, onChange, search, toOption, fetchDetail, detailItems, detailTitle, listPath, texts, emptyLabel, disabled, minWidth } = props
  const navigate = useNavigate()
  const [detail, setDetail] = useState<T | null>(null)
  const [detailErr, setDetailErr] = useState('')
  const [busy, setBusy] = useState(false)

  const openDetail = () => {
    setBusy(true)
    setDetailErr('')
    fetchDetail()
      .then((d) => { if (d) setDetail(d); else setDetailErr(texts.detailFail) })
      .catch(() => setDetailErr(texts.detailFail))
      .finally(() => setBusy(false))
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <ResourcePicker
        value={value}
        onChange={onChange}
        search={search}
        toOption={toOption}
        ariaLabel={texts.aria}
        emptyLabel={emptyLabel}
        searchPlaceholder={texts.placeholder}
        errorText={texts.loadFail}
        disabled={disabled}
        minWidth={minWidth}
      />
      <button type="button" className={LINK_BTN} disabled={!value || busy} onClick={openDetail}>
        {texts.viewDetail}
      </button>
      <button type="button" className={LINK_BTN} disabled={!value} onClick={() => navigate(listPath)}>
        {texts.jumpToList}
      </button>
      {detailErr && <span className="text-[11px] text-[var(--color-danger)]">{detailErr}</span>}
      {detail && (
        <DetailDrawer
          title={detailTitle}
          items={detailItems(detail)}
          onClose={() => setDetail(null)}
          closeText={texts.close}
        />
      )}
    </div>
  )
}
