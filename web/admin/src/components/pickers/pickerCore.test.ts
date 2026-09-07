// pickerCore 纯函数单测:双数据源合并、单/多选边界、分页选中、筛选参数组装、键盘导航、钉选回显、清空口径。
import { describe, expect, it } from 'vitest'
import {
  buildPickerQuery,
  canClearValue,
  mergeOptions,
  moveActive,
  pickSingleKey,
  toSelectionChips,
  togglePickKey,
  withPinnedValue,
} from './pickerCore'

const opt = (value: string, label?: string) => ({ value, label: label ?? value })

describe('mergeOptions 双数据源合并', () => {
  it('空值选项 + 静态选项按序拼接', () => {
    const out = mergeOptions([opt('', '全部')], [opt('a'), opt('b')])
    expect(out.map((o) => o.value)).toEqual(['', 'a', 'b'])
  })

  it('钉选选项优先,检索结果按 value 去重', () => {
    const out = mergeOptions([opt('p1', '钉选')], [opt('p1', '重复'), opt('p2')])
    expect(out).toHaveLength(2)
    expect(out[0]).toEqual(opt('p1', '钉选'))
    expect(out[1]).toEqual(opt('p2'))
  })

  it('容忍 undefined 组(调用方未传钉选/空值)', () => {
    const out = mergeOptions(undefined, undefined, [opt('x')])
    expect(out.map((o) => o.value)).toEqual(['x'])
  })

  it('全空入参返回空数组', () => {
    expect(mergeOptions(undefined, [])).toEqual([])
  })
})

describe('单/多选边界', () => {
  it('单选:重置为当前项', () => {
    expect(pickSingleKey('5')).toEqual(['5'])
  })

  it('单选:空串视为清空', () => {
    expect(pickSingleKey('')).toEqual([])
  })

  it('多选:勾选追加且保持顺序', () => {
    expect(togglePickKey(['a'], 'b', true)).toEqual(['a', 'b'])
  })

  it('多选:重复勾选不产生重复项', () => {
    expect(togglePickKey(['a', 'b'], 'b', true)).toEqual(['a', 'b'])
  })

  it('多选:取消勾选移除且不影响他项', () => {
    expect(togglePickKey(['a', 'b', 'c'], 'b', false)).toEqual(['a', 'c'])
  })

  it('多选:取消未勾选项返回原集合', () => {
    const cur = ['a']
    expect(togglePickKey(cur, 'x', false)).toEqual(cur)
  })
})

describe('分页选中与回显', () => {
  it('跨页选择顺序在多页切换后保持', () => {
    let selected: string[] = []
    // 第 1 页勾选 a、b
    selected = togglePickKey(selected, 'a', true)
    selected = togglePickKey(selected, 'b', true)
    // 翻到第 2 页勾选 c,回到第 1 页取消 a
    selected = togglePickKey(selected, 'c', true)
    selected = togglePickKey(selected, 'a', false)
    expect(selected).toEqual(['b', 'c'])
  })

  it('回显:按 key 序列映射 label,索引缺失项剔除', () => {
    const index = new Map([['a', '甲'], ['c', '丙']])
    const chips = toSelectionChips(['a', 'b', 'c'], index)
    expect(chips).toEqual([{ key: 'a', label: '甲' }, { key: 'c', label: '丙' }])
  })

  it('回显:空选中集返回空数组', () => {
    expect(toSelectionChips([], new Map([['a', '甲']]))).toEqual([])
  })
})

describe('buildPickerQuery 分页与筛选项组装', () => {
  it('page 转 offset,limit 直传', () => {
    expect(buildPickerQuery({ keyword: '', filters: {}, page: 3, pageSize: 20 })).toEqual({
      limit: 20,
      offset: 40,
    })
  })

  it('非法页码钳位为 1', () => {
    expect(buildPickerQuery({ keyword: '', filters: {}, page: 0, pageSize: 10 }).offset).toBe(0)
    expect(buildPickerQuery({ keyword: '', filters: {}, page: -2, pageSize: 10 }).offset).toBe(0)
  })

  it('关键字 trim 后非空才出参', () => {
    const q = buildPickerQuery({ keyword: '  张三  ', filters: {}, page: 1, pageSize: 10 })
    expect(q.keyword).toBe('张三')
    expect(buildPickerQuery({ keyword: '   ', filters: {}, page: 1, pageSize: 10 }).keyword).toBeUndefined()
  })

  it('筛选项空值(全部)剔除,非空透传', () => {
    const q = buildPickerQuery({ keyword: 'x', filters: { status: '', region: '  ', type: 'vip' }, page: 1, pageSize: 10 })
    expect(q.status).toBeUndefined()
    expect(q.region).toBeUndefined()
    expect(q.type).toBe('vip')
  })
})

describe('moveActive 键盘上下移动', () => {
  it('向下移动跳过禁用项', () => {
    expect(moveActive(3, 0, 1, (i) => i === 1)).toBe(2)
  })

  it('向上从首项循环回绕到末项', () => {
    expect(moveActive(3, 0, -1)).toBe(2)
  })

  it('向下从末项循环回绕到首项', () => {
    expect(moveActive(3, 2, 1)).toBe(0)
  })

  it('未初始化(-1):向下落首项,向上落末项', () => {
    expect(moveActive(3, -1, 1)).toBe(0)
    expect(moveActive(3, -1, -1)).toBe(2)
  })

  it('空列表返回 -1', () => {
    expect(moveActive(0, -1, 1)).toBe(-1)
  })

  it('全部禁用保持原位不动', () => {
    expect(moveActive(2, 1, 1, () => true)).toBe(1)
  })
})

describe('withPinnedValue 已选回显钉选', () => {
  it('已选值缺失时末尾追加合成选项(value 兼作 label)', () => {
    const out = withPinnedValue([opt('a')], 'x')
    expect(out).toHaveLength(2)
    expect(out[1]).toEqual({ value: 'x', label: 'x' })
  })

  it('已选值存在或值为空时不追加', () => {
    expect(withPinnedValue([opt('a')], 'a')).toHaveLength(1)
    expect(withPinnedValue([], '')).toHaveLength(0)
  })
})

describe('canClearValue 清空按钮可见性口径', () => {
  it('clearable 开启且未禁用且有值才可清', () => {
    expect(canClearValue(true, false, 'a')).toBe(true)
  })

  it('未开启/禁用/空值均不可清', () => {
    expect(canClearValue(undefined, false, 'a')).toBe(false)
    expect(canClearValue(true, true, 'a')).toBe(false)
    expect(canClearValue(true, false, '')).toBe(false)
  })
})
