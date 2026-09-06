// 建档/编辑纯逻辑用例:必填校验、载荷组装(契约 POST /assets、PUT /assets/:assetId)。
import { describe, expect, it } from 'vitest'
import { buildCreatePayload, buildEditPayload, emptyForm, formErrOf, scrapConfirmErr, scrapReasonErr } from './logic'
import type { AssetRow } from '../types'

const row = (over: Partial<AssetRow>): AssetRow => ({
  assetId: 1, assetCode: 'A-20260001', batchId: 9, legalEntityId: 1, legalEntityName: 'E',
  tagId: 0, addressId: 0, regionId: 0, regionName: '', type: '', modelId: 0, status: 'IN_STOCK',
  ...over,
})

describe('formErrOf', () => {
  it('批次必填', () => {
    expect(formErrOf({ ...emptyForm })).toBe('batch')
    expect(formErrOf({ ...emptyForm, type: 'ONU' })).toBe('batch')
  })
  it('类型与型号至少其一', () => {
    expect(formErrOf({ ...emptyForm, batchId: 3 })).toBe('type')
    expect(formErrOf({ ...emptyForm, batchId: 3, type: ' ONU ' })).toBe('')
    expect(formErrOf({ ...emptyForm, batchId: 3, modelId: 5 })).toBe('')
  })
})

describe('buildCreatePayload', () => {
  it('可选项未填不进载荷,type 去首尾空白', () => {
    expect(buildCreatePayload({ batchId: 3, modelId: 0, type: ' ONU ', tagId: 0 }))
      .toEqual({ batchId: 3, type: 'ONU' })
  })
  it('选中型号只传 modelId,type 由后端按 category 派生', () => {
    expect(buildCreatePayload({ batchId: 3, modelId: 7, type: '', tagId: 11 }))
      .toEqual({ batchId: 3, modelId: 7, tagId: 11 })
  })
})

describe('buildEditPayload', () => {
  it('未改动字段不传', () => {
    const origin = row({ batchId: 9, modelId: 7, type: 'ONU', tagId: 4 })
    expect(buildEditPayload({ batchId: 9, modelId: 7, type: 'ONU', tagId: 4 }, origin)).toEqual({})
  })
  it('换绑标签传新 tagId,解绑传 0', () => {
    expect(buildEditPayload({ ...emptyForm, batchId: 9, tagId: 8 }, row({ tagId: 4 }))).toEqual({ tagId: 8 })
    expect(buildEditPayload({ ...emptyForm, batchId: 9, tagId: 0 }, row({ tagId: 4 }))).toEqual({ tagId: 0 })
  })
  it('批次仅 IN_STOCK 态提交,其余状态改动被丢弃', () => {
    expect(buildEditPayload({ ...emptyForm, batchId: 12 }, row({ batchId: 9, status: 'DEPLOYED' }))).toEqual({})
    expect(buildEditPayload({ ...emptyForm, batchId: 12 }, row({ batchId: 9 }))).toEqual({ batchId: 12 })
  })
  it('未挂型号时改类型传 type', () => {
    expect(buildEditPayload({ ...emptyForm, batchId: 9, type: '路由器' }, row({ type: 'ONU' })))
      .toEqual({ type: '路由器' })
  })
})

describe('scrapReasonErr', () => {
  it('报废原因必填且不超 64 字', () => {
    expect(scrapReasonErr('')).toBe(true)
    expect(scrapReasonErr('   ')).toBe(true)
    expect(scrapReasonErr('x'.repeat(65))).toBe(true)
    expect(scrapReasonErr('外壳开裂')).toBe(false)
    expect(scrapReasonErr('x'.repeat(64))).toBe(false)
  })
})

describe('scrapConfirmErr', () => {
  const v = (code: string, sn: string, tagNo: string) => ({ code, sn, tagNo })
  it('资产编码恒必填', () => {
    expect(scrapConfirmErr(v('', 'S', 'T'), true, true)).toBe('code')
    expect(scrapConfirmErr(v('  ', 'S', 'T'), true, true)).toBe('code')
  })
  it('有 SN:confirmSn 必填', () => {
    expect(scrapConfirmErr(v('A', '', 'T'), true, true)).toBe('sn')
  })
  it('无 SN:confirmSn 须空串(动态规则,输入框不渲染时的兜底)', () => {
    expect(scrapConfirmErr(v('A', 'SN-X', ''), false, false)).toBe('sn')
  })
  it('已绑标签:confirmTagNo 必填', () => {
    expect(scrapConfirmErr(v('A', 'S', ''), true, true)).toBe('tagNo')
  })
  it('未绑标签:confirmTagNo 须空串', () => {
    expect(scrapConfirmErr(v('A', '', 'T-X'), false, false)).toBe('tagNo')
  })
  it('三要素齐备(有/无 两态)均通过', () => {
    expect(scrapConfirmErr(v('A', 'S', 'T'), true, true)).toBe('')
    expect(scrapConfirmErr(v('A', '', ''), false, false)).toBe('')
  })
})