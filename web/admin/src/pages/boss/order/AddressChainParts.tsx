// 内联建址弹层展示子件:面包屑/归属警示条/待治理黄标/链汇总/命名参照。
// 从 AddressChainDrawer 拆出独立文件,守 300 行红线。
import { useState } from 'react'
import { ReviewBadge } from './AddressChainBadge'
import type { AddressRow } from '../../base/address/AddressGeoDrawer'

// 命名参照:同层已有节点名常显(防「3栋/3号楼」并存);点击即复用;超 5 个折叠可展开。
const SIBLING_LIMIT = 5

export function SiblingHint({ nodes, pickText, moreText, onPick }: {
  nodes: AddressRow[]
  pickText: string
  moreText: string
  onPick: (n: AddressRow) => void
}) {
  const [open, setOpen] = useState(false)
  if (nodes.length === 0) return null
  const shown = open ? nodes : nodes.slice(0, SIBLING_LIMIT)
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="shrink-0 text-[11px] text-[var(--shell-group-title)]">
        {open ? moreText.replace('{count}', String(nodes.length)) : pickText}
      </span>
      {shown.map((n) => (
        <button key={n.id} type="button"
          className="inline-flex cursor-pointer items-center rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-1.5 py-0.5 text-[11px] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)]"
          onClick={() => onPick(n)}>{n.name}</button>
      ))}
      {nodes.length > SIBLING_LIMIT && (
        <button type="button"
          className="cursor-pointer border-none bg-none px-1 text-[11px] text-[var(--color-brand-gold-500)] hover:underline"
          onClick={() => setOpen(!open)}>
          {open ? '−' : moreText.replace('{count}', String(nodes.length))}
        </button>
      )}
    </div>
  )
}

export interface ChainStage {
  id?: number
  name: string
  needsReview?: boolean
}

// 面包屑:已完成层级可点击回退,回退丢弃其后层级;当前层级高亮,未到层级灰显。
export function ChainCrumb({ stages, active, labels, reviewText, onJump }: {
  stages: (ChainStage | null)[]
  active: number
  labels: string[]
  reviewText: string
  onJump: (stage: number) => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-1.5 text-[12px]">
      {labels.map((label, i) => {
        const done = stages[i]
        const isCurrent = i === active
        const clickable = done != null && i < active
        return (
          <span key={label} className="inline-flex items-center gap-1.5">
            {i > 0 && <span className="text-[var(--shell-group-title)]">/</span>}
            <button
              type="button"
              disabled={!clickable}
              onClick={() => clickable && onJump(i)}
              className={'inline-flex items-center gap-1 rounded-sm border-none bg-none px-1 py-0.5 text-[12px] ' + (clickable
                ? 'cursor-pointer text-[var(--color-brand-gold-500)] hover:underline'
                : isCurrent
                  ? 'cursor-default font-semibold text-[var(--shell-content-text)]'
                  : 'cursor-default text-[var(--shell-group-title)]')}>
              {done ? done.name : label}
              {done?.needsReview && <ReviewBadge text={reviewText} />}
            </button>
          </span>
        )
      })}
    </div>
  )
}

// 归属警示条:建址响应回传 legalEntityId 后常驻展示;兜底场景黄色警示但不拦提交。
// fallback=true 时主句直出三要素(发生什么/影响什么/接下来什么),去向句另起一行;不拦提交。
export function OwnerWarningBar({ entity, fallback, ownerText, fallbackText, fallbackHint }: {
  entity: string
  fallback: boolean
  ownerText: string
  fallbackText: string
  fallbackHint: string
}) {
  return (
    <div className={'rounded-sm border px-3 py-2 text-[12px] ' + (fallback
      ? 'border-[color-mix(in_srgb,var(--color-warning)_40%,transparent)] bg-[color-mix(in_srgb,var(--color-warning)_10%,transparent)] text-[var(--color-warning)]'
      : 'border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-content-text)]')}>
      <div>{fallback ? fallbackText : ownerText.replace('{entity}', entity)}</div>
      {fallback && <div className="mt-0.5">{fallbackHint}</div>}
    </div>
  )
}

// 链汇总:建址成功后的完整路径,逐级标注待治理黄标(与治理队列同口径)。
export function ChainSummary({ stages, fullPath, reviewText }: {
  stages: ChainStage[]
  fullPath: string
  reviewText: string
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="text-[13px] font-medium break-all text-[var(--shell-content-text)]">{fullPath}</div>
      <div className="flex flex-wrap items-center gap-1.5">
        {stages.map((s) => (
          <span key={`${s.name}-${s.id ?? 'new'}`}
            className="inline-flex items-center gap-1 rounded-sm bg-[var(--shell-menu-hover-bg)] px-1.5 py-0.5 text-[11px] text-[var(--shell-content-text)]">
            {s.name}
            {s.needsReview && <ReviewBadge text={reviewText} />}
          </span>
        ))}
      </div>
    </div>
  )
}
