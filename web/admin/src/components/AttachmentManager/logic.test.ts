import { describe, expect, it } from 'vitest'
import { fmtBytes, oversizeFiles, toListQuery, toggleSelection } from './logic'

describe('fmtBytes', () => {
  it('renders small sizes in B', () => {
    expect(fmtBytes(0)).toBe('0 B')
    expect(fmtBytes(512)).toBe('512 B')
  })
  it('renders KB/MB with one decimal below 100', () => {
    expect(fmtBytes(2048)).toBe('2.0 KB')
    expect(fmtBytes(5 * 1024 * 1024)).toBe('5.0 MB')
  })
  it('rounds up to integer at large values and guards invalid', () => {
    expect(fmtBytes(250 * 1024 * 1024)).toBe('250 MB')
    expect(fmtBytes(-1)).toBe('—')
    expect(fmtBytes(Number.NaN)).toBe('—')
  })
})

describe('toListQuery', () => {
  it('maps page to offset and keeps limit', () => {
    expect(toListQuery({ uploaderType: '', uploaderId: null, keyword: '', page: 3, pageSize: 20 }))
      .toEqual({ limit: 20, offset: 40 })
  })
  it('trims keyword and includes it when non-empty', () => {
    const q = toListQuery({ uploaderType: '', uploaderId: null, keyword: ' a ', page: 1, pageSize: 10 })
    expect(q.keyword).toBe('a')
    expect(toListQuery({ uploaderType: '', uploaderId: null, keyword: '   ', page: 1, pageSize: 10 }).keyword)
      .toBeUndefined()
  })
  it('emits uploader pair only when type and id both present', () => {
    const both = toListQuery({ uploaderType: 'worker', uploaderId: 9, keyword: '', page: 1, pageSize: 10 })
    expect(both.uploaderType).toBe('worker')
    expect(both.uploaderId).toBe(9)
    const typeOnly = toListQuery({ uploaderType: 'worker', uploaderId: null, keyword: '', page: 1, pageSize: 10 })
    expect(typeOnly.uploaderType).toBeUndefined()
    const idOnly = toListQuery({ uploaderType: '', uploaderId: 5, keyword: '', page: 1, pageSize: 10 })
    expect(idOnly.uploaderId).toBeUndefined()
  })
})

describe('toggleSelection', () => {
  it('appends without duplicates and removes', () => {
    expect(toggleSelection([1, 2], 3, true)).toEqual([1, 2, 3])
    expect(toggleSelection([1, 2], 2, true)).toEqual([1, 2])
    expect(toggleSelection([1, 2], 1, false)).toEqual([2])
  })
})

describe('oversizeFiles', () => {
  it('flags files above 32MB only', () => {
    const big = new File(['x'], 'big.bin')
    Object.defineProperty(big, 'size', { value: 33 * 1024 * 1024 })
    const small = new File(['x'], 'small.bin') // size 1
    expect(oversizeFiles([big, small])).toEqual(['big.bin'])
  })
})
