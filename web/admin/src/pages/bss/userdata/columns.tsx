// 用户端配置页 per-tab 列定义。字段名即后端 SQL AS 别名契约
// (internal/domain/customer/userdata/pg_lists.go;此内部配置页未登记 fields.md,以 SQL 别名为准)。
// 布尔配置映射 StatusTag userdata.active/disabled;增值服务/优惠券状态字符串直传。
import type { ReactNode } from 'react'
import type { ColumnDef } from '../../../components/business/data-table'
import { StatusTag } from '../../../components/StatusTag'
import { ActionLink } from '../../../components/business/page-head'
import { fmtFee, fmtTime } from '../../../lib/format'
import type { Row, TabDef } from './tabs'

export interface Labels {
  cols: Record<string, string>
  channels: Record<string, string>
  nameCol: string
  statusCol: string
  opCol: string
  disable: string
  on: string
  off: string
  list: string
  unlist: string
}

const dash = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : String(v))
const money = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : fmtFee(Number(v)))

/** 布尔 → active/disabled;字符串状态原样交给 StatusTag(注册表外灰色兜底)。 */
function Status({ v }: { v: unknown }) {
  const value = typeof v === 'boolean' ? (v ? 'active' : 'disabled') : String(v ?? '')
  return <StatusTag domain="userdata" value={value} />
}

/** 主键列:缺列时渲染 —missing 而非 undefined 字面值(postmortem 0002 纵深防御)。 */
function IdCell({ def, row }: { def: TabDef; row: Row }) {
  const v = row[def.idKey]
  if (v === undefined || v === null || v === '') return <span title={`缺主键列 ${def.idKey}`}>—missing</span>
  return <>{String(v)}</>
}

const col = (key: string, label: string, render?: (r: Row) => ReactNode): ColumnDef => ({ key, label, render })

/** 每 tab 列组;操作列仅在 def.action 存在时渲染,与后端动作路由一一对应。 */
export function columnsFor(def: TabDef, lb: Labels, onAct: (r: Row) => void): ColumnDef[] {
  const c = lb.cols
  const id = col('id', 'ID', (r) => <IdCell def={def} row={r} />)
  const op: ColumnDef[] = def.action
    ? [col('op', lb.opCol, (r) => (
        <ActionLink
          testId={`userdata-op-${def.key}`}
          label={opLabel(def, r, lb)}
          onClick={() => onAct(r)}
        />
      ))]
    : []
  switch (def.key) {
    case 'notify':
      return [
        id,
        col('customer', c.customer, (r) => dash(r.customerName)),
        col('business', c.business, (r) => <Status v={r.business} />),
        col('marketing', c.marketing, (r) => <Status v={r.marketing} />),
        col('channel', c.channel, (r) => lb.channels[String(r.channel)] ?? dash(r.channel)),
      ]
    case 'addons':
      return [
        id,
        col('name', lb.nameCol, (r) => dash(r.name)),
        col('price', c.price, (r) => money(r.price)),
        col('status', lb.statusCol, (r) => <Status v={r.status} />),
        col('subscriberCount', c.subscriberCount, (r) => dash(r.subscriberCount)),
        ...op,
      ]
    case 'coupons':
      return [
        id,
        col('customer', c.customer, (r) => dash(r.customerId)),
        col('name', lb.nameCol, (r) => dash(r.name)),
        col('amount', c.amount, (r) => money(r.amount)),
        col('status', lb.statusCol, (r) => <Status v={r.status} />),
        col('expireAt', c.expireAt, (r) => fmtTime(r.expireAt as string | undefined)),
        ...op,
      ]
    case 'topup':
      return [
        id,
        col('amount', c.amount, (r) => money(r.amount)),
        col('bonus', c.bonus, (r) => money(r.bonus)),
        col('status', lb.statusCol, (r) => <Status v={r.active} />),
      ]
    case 'faqs':
      return [
        id,
        col('category', c.category, (r) => dash(r.category)),
        col('question', c.question, (r) => dash(r.question)),
        col('status', lb.statusCol, (r) => <Status v={r.active} />),
        ...op,
      ]
    case 'guides':
      return [
        id,
        col('title', c.title, (r) => dash(r.title)),
        col('category', c.category, (r) => dash(r.category)),
        col('status', lb.statusCol, (r) => <Status v={r.active} />),
        ...op,
      ]
    case 'invite':
      return [
        id,
        col('inviteLink', c.inviteLink, (r) => dash(r.inviteLink)),
        col('rewardAmount', c.rewardAmount, (r) => money(r.rewardAmount)),
        col('status', lb.statusCol, (r) => <Status v={r.active} />),
      ]
    default:
      return [id]
  }
}

/** 动作动词反映行当前态(antd link-button 语义):增值服务按上下架,布尔配置按启用/停用。 */
export function opLabel(def: TabDef, r: Row, lb: Labels): string {
  if (def.action === 'disable') return lb.disable
  if (def.key === 'addons') return r.status === 'on' ? lb.unlist : lb.list
  return r.active ? lb.off : lb.on
}
