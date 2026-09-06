// 型号字典/批次管理组件级用例:必填校验、载荷组装、启停显隐(仅在用显停用、仅停用显启用)。
import { describe, expect, it } from 'vitest'
import {
  batchFormErr, buildBatchPayload, buildModelPayload, canDisableModel,
  canEnableModel, emptyBatchForm, emptyModelForm, modelFormErr,
} from './dictLogic'

describe('modelFormErr / buildModelPayload', () => {
  it('型号名与类别必填,其余可空', () => {
    expect(modelFormErr({ ...emptyModelForm })).toBe('eModelRequired')
    expect(modelFormErr({ vendor: '华为', model: 'HN-ONU', category: '', partNumber: '' })).toBe('eModelRequired')
    expect(modelFormErr({ vendor: '', model: 'HN-ONU', category: 'ONU', partNumber: '' })).toBe('')
  })
  it('载荷组装去空白且四键齐备', () => {
    expect(buildModelPayload({ vendor: ' 华为 ', model: ' HN-ONU ', category: ' ONU ', partNumber: ' P1 ' }))
      .toEqual({ vendor: '华为', model: 'HN-ONU', category: 'ONU', partNumber: 'P1' })
  })
  it('启停显隐:在用仅可停用,停用仅可启用', () => {
    expect(canDisableModel({ isActive: true })).toBe(true)
    expect(canDisableModel({ isActive: false })).toBe(false)
    expect(canEnableModel({ isActive: false })).toBe(true)
    expect(canEnableModel({ isActive: true })).toBe(false)
  })
})

describe('batchFormErr / buildBatchPayload', () => {
  it('批次编码必填', () => {
    expect(batchFormErr({ ...emptyBatchForm })).toBe('eBatchRequired')
    expect(batchFormErr({ code: ' ', name: '' })).toBe('eBatchRequired')
    expect(batchFormErr({ code: 'RK-001', name: '' })).toBe('')
  })
  it('载荷组装仅含 code/name 且去空白', () => {
    expect(buildBatchPayload({ code: ' RK-9 ', name: ' 九月入库 ' })).toEqual({ code: 'RK-9', name: '九月入库' })
    expect(Object.keys(buildBatchPayload({ code: 'A', name: 'B' })).sort()).toEqual(['code', 'name'])
  })
})
