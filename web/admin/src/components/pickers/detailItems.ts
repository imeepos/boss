// 选择器详情字段映射:纯函数,文案由调用方 i18n 传入,便于单测。
import type { CustomerItem, UserDetail, WorkerItem } from '../../api/pickers'
import type { DetailItem } from '../../pages/org/shared'
import { fmtTime } from '../../lib/format'

export interface WorkerDetailTexts {
  name: string; staffNo: string; phone: string; group: string
  region: string; status: string; onDuty: string; left: string; joinedAt: string
}

export function workerDetailItems(w: WorkerItem, t: WorkerDetailTexts): DetailItem[] {
  return [
    { k: t.name, v: w.name },
    { k: t.staffNo, v: w.staffNo },
    { k: t.phone, v: w.phone },
    { k: t.group, v: w.groupId ? `#${w.groupId}` : '—' },
    { k: t.region, v: w.regionId ? `#${w.regionId}` : '—' },
    { k: t.status, v: w.status === 1 ? t.onDuty : t.left },
    { k: t.joinedAt, v: fmtTime(w.joinedAt) },
  ]
}

export interface CustomerDetailTexts {
  code: string; name: string; phone: string; idType: string; idNo: string
  realNameStatus: string; serviceStatus: string; region: string; createdAt: string
}

export function customerDetailItems(c: CustomerItem, t: CustomerDetailTexts): DetailItem[] {
  return [
    { k: t.code, v: c.customerCode },
    { k: t.name, v: c.name },
    { k: t.phone, v: c.phone },
    { k: t.idType, v: c.idType },
    { k: t.idNo, v: c.idNo },
    { k: t.realNameStatus, v: c.realNameStatus },
    { k: t.serviceStatus, v: c.serviceStatus },
    { k: t.region, v: c.regionName || (c.regionId ? `#${c.regionId}` : '—') },
    { k: t.createdAt, v: fmtTime(c.createdAt) },
  ]
}

export interface UserDetailTexts {
  name: string; phone: string; plan: string; balance: string; arrears: string
  activeOrders: string; realNameStatus: string; serviceStatus: string; autoPay: string; on: string; off: string
}

export function userDetailItems(u: UserDetail, t: UserDetailTexts): DetailItem[] {
  return [
    { k: t.name, v: u.name },
    { k: t.phone, v: u.phone },
    { k: t.plan, v: u.planName },
    { k: t.balance, v: String(u.balance) },
    { k: t.arrears, v: String(u.arrearsAmount) },
    { k: t.activeOrders, v: String(u.activeOrders) },
    { k: t.realNameStatus, v: u.realNameStatus },
    { k: t.serviceStatus, v: u.serviceStatus },
    { k: t.autoPay, v: u.autoPay ? t.on : t.off },
  ]
}
