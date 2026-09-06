// 标签页纯逻辑:表单校验/载荷组装/操作列状态显隐(契约 POST /tags、/:id/disable|enable、/unbind)。
// 状态枚举 terms.md §4:UNBOUND/BOUND/DISABLED。
import type { TagRow } from '../types'

export interface TagFormState {
  tagNo: string
  epcCode: string
  band: string
}

export const emptyTagForm: TagFormState = { tagNo: '', epcCode: '', band: '' }

export function tagFormErr(form: TagFormState): string {
  if (!form.tagNo.trim() || !form.epcCode.trim()) return 'eTagNo'
  return ''
}

export function buildTagPayload(form: TagFormState) {
  return { tagNo: form.tagNo.trim(), epcCode: form.epcCode.trim(), band: form.band.trim() }
}

// 操作列显隐:禁用对非 DISABLED 开放;启用仅 DISABLED;解绑仅已绑定(BOUND)。
export const canDisableTag = (status: string): boolean => status !== 'DISABLED'
export const canEnableTag = (status: string): boolean => status === 'DISABLED'
export const canUnbindTag = (status: string): boolean => status === 'BOUND'

export function tagActionsOf(row: Pick<TagRow, 'status'>): string[] {
  const acts: string[] = []
  if (canDisableTag(row.status)) acts.push('disable')
  if (canEnableTag(row.status)) acts.push('enable')
  if (canUnbindTag(row.status)) acts.push('unbind')
  acts.push('events')
  return acts
}
