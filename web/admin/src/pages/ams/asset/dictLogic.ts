// 型号字典/批次管理纯逻辑:表单校验、载荷组装、启停显隐(契约 POST/PUT /asset-models、/asset-batches)。
import type { AssetModelRow } from '../types'

export interface ModelFormState {
  vendor: string
  model: string
  category: string
  partNumber: string
}

export const emptyModelForm: ModelFormState = { vendor: '', model: '', category: '', partNumber: '' }

export function modelFormErr(form: ModelFormState): string {
  if (!form.model.trim() || !form.category.trim()) return 'eModelRequired'
  return ''
}

// 编辑复用同载荷契约:PUT /asset-models/{id} 与 POST 同形(明细可整体覆盖)。
export function buildModelPayload(form: ModelFormState) {
  return {
    vendor: form.vendor.trim(),
    model: form.model.trim(),
    category: form.category.trim(),
    partNumber: form.partNumber.trim(),
  }
}

// 启停显隐:停用仅对在用型号开放;启用仅对已停用型号开放(建档下拉只取 isActive)。
export const canDisableModel = (m: Pick<AssetModelRow, 'isActive'>): boolean => m.isActive
export const canEnableModel = (m: Pick<AssetModelRow, 'isActive'>): boolean => !m.isActive

export interface BatchFormState {
  code: string
  name: string
}

export const emptyBatchForm: BatchFormState = { code: '', name: '' }

export function batchFormErr(form: BatchFormState): string {
  if (!form.code.trim()) return 'eBatchRequired'
  return ''
}

export function buildBatchPayload(form: BatchFormState) {
  return { code: form.code.trim(), name: form.name.trim() }
}
