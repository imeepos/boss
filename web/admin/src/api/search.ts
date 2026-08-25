// 聚合搜索 API:契约 api/openapi/admin/search.yaml(GET /search?keyword=)。
// 返回四域分组结果,字段复用各域列表契约(customer/userdata/worker/order.yaml)。
import { apiFetch } from './client'

export type SearchDomain = 'customer' | 'user' | 'worker' | 'order'

export interface CustomerHit {
  id: number
  customerCode: string
  name: string
  phone: string
  serviceStatus: string
}

export interface UserHit {
  customerId: number
  name: string
  phone: string
  loginName: string
  planName: string
  serviceStatus: string
}

export interface WorkerHit {
  id: number
  staffNo: string
  name: string
  phone: string
  status: number
}

export interface OrderHit {
  id: number
  orderNo: string
  customer: string
  product: string
  status: string
  stage: number
}

export type SearchHit = CustomerHit | UserHit | WorkerHit | OrderHit

export interface SearchGroup {
  domain: SearchDomain
  items: SearchHit[]
}

/** 聚合检索:keyword 为空不请求(后端 422)。 */
export function aggregateSearch(keyword: string): Promise<SearchGroup[]> {
  return apiFetch<{ groups: SearchGroup[] }>('/search', { query: { keyword } }).then((d) => d?.groups ?? [])
}
