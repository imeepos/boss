// 资产建档/编辑纯逻辑:必填校验与载荷组装(契约 POST /assets、PUT /assets/:assetId)。
// 载荷口径:未改动/未选字段不传;选中型号后类型由 model_id->category 权威派生,不再传 type。
import type { AssetModelRow, AssetRow } from '../types'

export interface AssetFormState {
  batchId: number
  modelId: number
  type: string
  tagId: number
  sn: string
  mac: string
  loid: string
}

export const emptyForm: AssetFormState = { batchId: 0, modelId: 0, type: '', tagId: 0, sn: '', mac: '', loid: ''}

export type FormErr = '' | 'batch' | 'type'

// 建档/编辑共用校验:批次必填;类型与型号至少其一。'' = 通过。
export function formErrOf(f: AssetFormState): FormErr {
  if (!f.batchId) return 'batch'
  if (!f.modelId && !f.type.trim()) return 'type'
  return ''
}

// 型号下拉展示:厂商 - 型号名,厂商缺失退型号名,再退 #id。
export function modelLabel(m: AssetModelRow): string {
  const label = [m.vendor, m.model].filter(Boolean).join(' · ')
  return label || '#' + String(m.id)
}

// 表单类型的展示值:选中型号时取 category(权威),否则取手输值。
export function typeValue(f: AssetFormState, modelOf: (id: number) => AssetModelRow | undefined): string {
  const m = f.modelId ? modelOf(f.modelId) : undefined
  return m ? m.category : f.type
}

export interface CreatePayload {
  batchId: number
  modelId?: number
  type?: string
  tagId?: number
  sn?: string
  mac?: string
  loid?: string
}

export function buildCreatePayload(f: AssetFormState): CreatePayload {
  const p: CreatePayload = { batchId: f.batchId }
  if (f.modelId) p.modelId = f.modelId
  else if (f.type.trim()) p.type = f.type.trim()
  if (f.tagId) p.tagId = f.tagId
  // 身份三要素(P3-T2):空=不传,非空去首尾空格(服务端再校验/归一,空串存 NULL)。
  if (f.sn.trim()) p.sn = f.sn.trim()
  if (f.mac.trim()) p.mac = f.mac.trim()
  if (f.loid.trim()) p.loid = f.loid.trim()
  return p
}

export interface EditPayload {
  type?: string
  modelId?: number
  tagId?: number
  batchId?: number
  sn?: string
  mac?: string
  loid?: string
}

// 编辑载荷:仅传改动字段;tagId 允许传 0 表示解绑;批次仅 IN_STOCK 态提交;
// 身份三要素(P3-T2)有变才传,清空传空串(服务端语义=清除存 NULL)。
export function buildEditPayload(f: AssetFormState, origin: AssetRow): EditPayload {
  const p: EditPayload = {}
  if (f.batchId && f.batchId !== origin.batchId && origin.status === 'IN_STOCK') p.batchId = f.batchId
  if (f.modelId && f.modelId !== origin.modelId) p.modelId = f.modelId
  else if (!f.modelId && f.type.trim() && f.type.trim() !== origin.type) p.type = f.type.trim()
  if (f.tagId !== origin.tagId) p.tagId = f.tagId
  if (f.sn.trim() !== origin.sn) p.sn = f.sn.trim()
  if (f.mac.trim() !== origin.mac) p.mac = f.mac.trim()
  if (f.loid.trim() !== origin.loid) p.loid = f.loid.trim()
  return p
}

// 报废原因:必填且不超 64 字(契约 reason maxLength 64)。true = 不合法。
export function scrapReasonErr(reason: string): boolean {
  const t = reason.trim()
  return t.length === 0 || t.length > 64
}