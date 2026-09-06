// 建档/编辑纯逻辑用例:必填校验、载荷组装(契约 POST /assets、PUT /assets/:assetId)。
import { describe, expect, it } from 'vitest'
import { buildCreatePayload, buildEditPayload, emptyForm, formErrOf, scrapReasonErr } from './logic'
import type { AssetRow } from '../types'

const row = (over: Partial<AssetRow>): AssetRow => ({
  assetId: 1, assetCode: 'A-20260001', batchId: 9, legalEntityId: 1, legalEntityName: 'E',
  tagId: 0, addressId: 0, regionId: 0, regionName: '', type: '', modelId: 0, status: 'IN_STOCK',
  sn: '', mac: '', loid: '',
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
    expect(buildCreatePayload({ batchId: 3, modelId: 0, type: ' ONU ', tagId: 0, sn: '', mac: '', loid: '' }))
      .toEqual({ batchId: 3, type: 'ONU' })
  })
  it('选中型号只传 modelId,type 由后端按 category 派生', () => {
    expect(buildCreatePayload({ batchId: 3, modelId: 7, type: '', tagId: 11, sn: '', mac: '', loid: '' }))
      .toEqual({ batchId: 3, modelId: 7, tagId: 11 })
  })
  it('身份三要素非空才进载荷并去首尾空白(P3-T2)', () => {
    expect(buildCreatePayload({ batchId: 3, modelId: 0, type: '', tagId: 0, sn: ' SN1 ', mac: 'AA-BB-CC-DD-EE-01', loid: '' }))
      .toEqual({ batchId: 3, sn: 'SN1', mac: 'AA-BB-CC-DD-EE-01' })
  })
})

describe('buildEditPayload', () => {
  it('未改动字段不传', () => {
    const origin = row({ batchId: 9, modelId: 7, type: 'ONU', tagId: 4 })
    expect(buildEditPayload({ batchId: 9, modelId: 7, type: 'ONU', tagId: 4, sn: '', mac: '', loid: '' }, origin)).toEqual({})
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
  it('身份三要素有变才传,清空传空串(P3-T2 清除语义)', () => {
    const origin = row({ sn: 'SN-OLD', mac: 'AA:BB:CC:DD:EE:01' })
    expect(buildEditPayload({ ...emptyForm, sn: 'SN-NEW', mac: 'AA:BB:CC:DD:EE:01', loid: '' }, origin))
      .toEqual({ sn: 'SN-NEW' })
    expect(buildEditPayload({ ...emptyForm, sn: '  ', mac: 'AA:BB:CC:DD:EE:01', loid: '' }, origin))
      .toEqual({ sn: '' })
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