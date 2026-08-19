// org 组六页公共片段:页头、详情抽屉、分页文案装配。
// 已迁移到 components/business/，此处保留兼容导出。
import { Drawer } from '../../components/Drawer'
import type { Translations } from '../../i18n/types'

type PagerNs = Pick<
  Translations['pages']['company'],
  'rangeText' | 'prev' | 'next' | 'perPage' | 'jumpText' | 'pageUnit'
>

/** 装配 Pagination 所需文案(各 namespace 同名 key),可直接展开传给 Pagination。 */
export function pagerTexts(ns: PagerNs) {
  return {
    rangeText: ns.rangeText, prevText: ns.prev, nextText: ns.next,
    perPageText: ns.perPage, jumpText: ns.jumpText, pageUnitText: ns.pageUnit,
  }
}

/** 页头:标题 + 一行描述(契约口径)。 */
export function PageHead({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="mb-4">
      <h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{title}</h2>
      <p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{desc}</p>
    </div>
  )
}

export interface DetailItem { k: string; v: string }

/** 详情抽屉:只读 k-v 列表(替代原型 modal 详情)。 */
export function DetailDrawer({
  title, items, onClose, closeText,
}: { title: string; items: DetailItem[]; onClose: () => void; closeText: string }) {
  return (
    <Drawer
      title={title}
      onClose={onClose}
      footer={
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{closeText}</button>
      }
    >
      <div className="flex flex-col gap-2.5">
        {items.map((it) => (
          <div key={it.k} className="flex gap-3 text-[13px]">
            <span className="w-24 flex-none text-[var(--shell-group-title)]">{it.k}</span>
            <span className="break-all text-[var(--shell-content-text)]">{it.v || '—'}</span>
          </div>
        ))}
      </div>
    </Drawer>
  )
}

/** 空态行/块。 */
export function Empty({ text }: { text: string }) {
  return <div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{text}</div>
}