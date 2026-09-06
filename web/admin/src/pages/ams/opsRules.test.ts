import { describe, expect, it } from 'vitest'
import { canConfirmScrap, canConfirmUnbind, changedPairs } from './opsRules'

// C4 佐证:危险确认「原因必填」与报废「资产编码确认」逻辑单测。
describe('危险操作确认规则', () => {
  it('解绑:原因必填,空白/纯空格拒绝', () => {
    expect(canConfirmUnbind('')).toBe(false)
    expect(canConfirmUnbind('   ')).toBe(false)
    expect(canConfirmUnbind('换新回收')).toBe(true)
  })

  it('报废:原因必填且资产编码须精确匹配', () => {
    expect(canConfirmScrap('', 'A-20260001', 'A-20260001')).toBe(false)
    expect(canConfirmScrap('   ', 'A-20260001', 'A-20260001')).toBe(false)
    expect(canConfirmScrap('屏裂报废', '', 'A-20260001')).toBe(false)
    expect(canConfirmScrap('屏裂报废', 'A-20260002', 'A-20260001')).toBe(false)
    expect(canConfirmScrap('屏裂报废', ' A-20260001 ', 'A-20260001')).toBe(true)
  })
})

describe('changedPairs 只渲染实际变化键', () => {
  it('数组值渲染 键: 旧值 → 新值', () => {
    expect(changedPairs({ bound_asset_id: [2, 0] })).toEqual([{ key: 'bound_asset_id', from: '2', to: '0' }])
  })

  it('缺失/空 changed 不产出行;非数组只出新值', () => {
    expect(changedPairs(undefined)).toEqual([])
    expect(changedPairs(null)).toEqual([])
    expect(changedPairs({})).toEqual([])
    expect(changedPairs({ status: 'DEPLOYED' })).toEqual([{ key: 'status', from: '', to: 'DEPLOYED' }])
  })
})
