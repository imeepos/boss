// MultiSelect 纯函数回归:勾选切换与关键字过滤(事件选择器的选择语义)。
import { describe, expect, it } from 'vitest'
import { filterOptions, toggleValue, type MultiOption } from './MultiSelect'

const opts: MultiOption[] = [
  { value: 'order.stage.done', label: 'order.stage.done', description: 'Order stage advanced' },
  { value: 'order.activated', label: 'order.activated' },
]

describe('toggleValue', () => {
  it('未选则追加(保序)', () => {
    expect(toggleValue([], 'a')).toEqual(['a'])
    expect(toggleValue(['a'], 'b')).toEqual(['a', 'b'])
  })
  it('已选则移除且保序', () => {
    expect(toggleValue(['a', 'b', 'c'], 'b')).toEqual(['a', 'c'])
  })
  it('重复切换幂等', () => {
    expect(toggleValue(toggleValue(['a'], 'b'), 'b')).toEqual(['a'])
  })
})

describe('filterOptions', () => {
  it('空关键字返回全量', () => {
    expect(filterOptions(opts, '')).toHaveLength(2)
    expect(filterOptions(opts, '  ')).toHaveLength(2)
  })
  it('按 label 大小写不敏感前缀内匹配', () => {
    expect(filterOptions(opts, 'ORDER')).toEqual([opts[0], opts[1]])
    expect(filterOptions(opts, 'activated')).toEqual([opts[1]])
    expect(filterOptions(opts, 'nope')).toEqual([])
  })
})
