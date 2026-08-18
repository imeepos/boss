// org 组六页公共片段:页头、详情抽屉、分页文案装配。
import type { ReactNode } from 'react'
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
    <div className="org-page-head">
      <h2 className="org-page-title">{title}</h2>
      <p className="org-page-desc">{desc}</p>
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
        <button className="org-btn org-btn-primary" onClick={onClose}>{closeText}</button>
      }
    >
      <div className="org-detail-list">
        {items.map((it) => (
          <div key={it.k} className="org-detail-item">
            <span className="k">{it.k}</span>
            <span className="v">{it.v || '—'}</span>
          </div>
        ))}
      </div>
    </Drawer>
  )
}

/** 空态行/块。 */
export function Empty({ text }: { text: string }): ReactNode {
  return <div className="org-empty">{text}</div>
}
