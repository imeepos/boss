// 选择器数据源:用户/师傅/客户 三域 keyword 服务端检索 + 单档详情。
// 契约:api/openapi/admin/{userdata,worker,customer}.yaml;字段以 docs/contract/fields.md 为准。
import { apiFetch } from './client'

export interface WorkerItem {
  id: number
  staffNo: string
  name: string
  groupId: number
  regionId: number
  phone: string
  status: number // 1 在职 0 离职
  joinedAt: string
}

export interface CustomerItem {
  id: number
  customerCode: string
  name: string
  phone: string
  idType: string
  idNo: string
  realNameStatus: string
  serviceStatus: string
  addressId: number
  legalEntityId: number
  regionId: number
  regionName: string
  createdAt: string
}

export interface UserItem {
  customerId: number
  loginName: string
  name: string
  phone: string
  planName: string
  realNameStatus: string
  serviceStatus: string
  balance: number
  arrearsAmount: number
  activeOrders: number
  autoPay: boolean
}

/** 用户详情聚合(/users/{customerId} 返回 14 类子数据,此处仅取主档摘要)。 */
export interface UserDetail extends UserItem {
  addresses?: Array<{ id: number; addrCode?: string; detail?: string }>
  plans?: Array<{ productName?: string; status?: string }>
}

export function searchUsers(keyword: string): Promise<UserItem[] | null> {
  return apiFetch<{ items: UserItem[] }>('/users', { query: { keyword: keyword || undefined } }).then((d) => d?.items ?? [])
}

export function getUserDetail(customerId: string): Promise<UserDetail | null> {
  return apiFetch<UserDetail>(`/users/${encodeURIComponent(customerId)}`)
}

export function searchWorkers(keyword: string): Promise<WorkerItem[] | null> {
  return apiFetch<{ items: WorkerItem[] }>('/workers', { query: { keyword: keyword || undefined } }).then((d) => d?.items ?? [])
}

export function getWorkerDetail(workerId: string): Promise<WorkerItem | null> {
  return apiFetch<WorkerItem>(`/workers/${encodeURIComponent(workerId)}`)
}

export function searchCustomers(keyword: string): Promise<CustomerItem[] | null> {
  return apiFetch<{ items: CustomerItem[] }>('/customers', { query: { keyword: keyword || undefined } }).then((d) => d?.items ?? [])
}

export function getCustomerDetail(customerId: string): Promise<CustomerItem | null> {
  return apiFetch<CustomerItem>(`/customers/${encodeURIComponent(customerId)}`)
}
