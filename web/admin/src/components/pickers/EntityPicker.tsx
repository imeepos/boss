// 实体选择器:服务端检索下拉(基于 SimplePicker 小数据量基座)+ 选中后"详情/前往管理页"入口。
// 详情走各域单档接口(抽屉展示);跳转用 react-router 导航到对应管理页。
// W0-R3 已选回显:检索结果 label 增量缓存,页面重hydrate等缓存漏项时走详情接口兜底钉选,
// 触发器始终展示人类可读名称,不跌回裸内部编号。
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { SimplePicker } from './SimplePicker'
import { DetailDrawer } from '../business/detail-drawer'
import { echoPinFromCache, rememberOptionLabels } from './pickerCore'
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
  /** 展示一键清空按钮(值非空且未禁用时);清空即 onChange('')。 */
  clearable?: boolean
  /** 清空按钮 aria 文案(clearable 时由调用方 i18n 传入)。 */
  clearLabel?: string
}

const LINK_BTN = 'h-8 cursor-pointer rounded-sm border border-none bg-none px-2 text-xs whitespace-nowrap text-primary hover:underline disabled:cursor-not-allowed disabled:opacity-50'

export function EntityPicker<T>(props: EntityPickerProps<T>) {
  const { value, onChange, search, toOption, fetchDetail, detailItems, detailTitle, listPath, texts, emptyLabel, disabled, minWidth, clearable, clearLabel } = props
  const navigate = useNavigate()
  const [detail, setDetail] = useState<T | null>(null)
  const [detailErr, setDetailErr] = useState('')
  const [busy, setBusy] = useState(false)
  // 已选人类可读回显:label 缓存(检索结果增量喂入)+ 详情兜底,合成 pinnedOptions 下传基座。
  const labelCache = useRef(new Map<string, string>())
  const toOptionRef = useRef(toOption)
  toOptionRef.current = toOption
  const [echoPin, setEchoPin] = useState<DropdownOption | undefined>(undefined)
  useEffect(() => {
    setEchoPin(echoPinFromCache(labelCache.current, value))
    if (value === '' || labelCache.current.has(value)) return undefined
    let alive = true
    fetchDetail()
      .then((d) => {
        if (!alive || !d) return
        const opt = toOptionRef.current(d)
        if (!opt.label || opt.label === value) return
        labelCache.current.set(value, opt.label)
        setEchoPin({ value, label: opt.label })
        if (opt.value !== value) console.warn('[EntityPicker] 详情实体与已选值不一致: ' + value + ' ≠ ' + opt.value)
      })
      .catch((err) => {
        // 回显兜底失败不阻塞填表:留痕后交由基座 withPinnedValue 以原值回显。
        console.warn('[EntityPicker] 已选回显详情拉取失败: ' + value, err)
      })
    return () => { alive = false }
    // fetchDetail 捕获当轮渲染的 value,随 value 变化重放;仅 value 驱动。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

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
      <SimplePicker
        value={value}
        onChange={onChange}
        search={(kw) => search(kw).then((items) => {
          const opts = (items ?? []).map(toOption)
          rememberOptionLabels(labelCache.current, opts)
          return opts
        })}
        ariaLabel={texts.aria}
        placeholder={texts.placeholder}
        emptyLabel={emptyLabel}
        pinnedOptions={echoPin ? [echoPin] : undefined}
        searchPlaceholder={texts.placeholder}
        errorText={texts.loadFail}
        clearable={clearable}
        clearLabel={clearLabel}
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
