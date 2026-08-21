import { describe, expect, it } from 'vitest'
import {
  classifyAttachment, countByCategory, filterByCategory,
  fmtBytes, oversizeFiles, toListQuery, toggleSelection,
} from './logic'

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

describe('classifyAttachment', () => {
  it('classifies by MIME major type', () => {
    expect(classifyAttachment('image/png', 'anything.xyz')).toBe('image')
    expect(classifyAttachment('audio/mpeg', 'song')).toBe('audio')
    expect(classifyAttachment('video/mp4', 'movie')).toBe('video')
    expect(classifyAttachment('application/pdf', 'doc')).toBe('pdf')
  })
  it('classifies by explicit MIME table', () => {
    expect(classifyAttachment('application/zip', 'unknown')).toBe('archive')
    expect(classifyAttachment('application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'x'))
      .toBe('document')
    expect(classifyAttachment('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', 'x'))
      .toBe('spreadsheet')
  })
  it('classifies text/* as code (CSV gets spreadsheet via MIME table)', () => {
    expect(classifyAttachment('text/plain', 'README')).toBe('code')
    expect(classifyAttachment('text/csv', 'a.csv')).toBe('spreadsheet')
  })
  it('falls back to extension when MIME is empty/unknown', () => {
    expect(classifyAttachment('', 'photo.JPG')).toBe('image')
    expect(classifyAttachment('', '合同扫描件.PDF')).toBe('pdf')
    expect(classifyAttachment('application/octet-stream', 'data.xlsx')).toBe('spreadsheet')
    expect(classifyAttachment('application/octet-stream', 'backup.tar.gz')).toBe('archive')
    expect(classifyAttachment('application/octet-stream', 'config.json')).toBe('code')
  })
  it('handles Chinese filenames and URL-stripped names', () => {
    expect(classifyAttachment('', '现场照片.jpg?token=abc')).toBe('image')
    expect(classifyAttachment('', '工单记录.json')).toBe('code')
    expect(classifyAttachment('', '数据导出.xlsx')).toBe('spreadsheet')
  })
  it('returns other for unknown extension or empty', () => {
    expect(classifyAttachment('', 'README')).toBe('other')
    expect(classifyAttachment('', 'file.unknownext')).toBe('other')
    expect(classifyAttachment('', '')).toBe('other')
  })
})

describe('filterByCategory', () => {
  const items = [
    { id: 1, contentType: 'image/png', fileName: 'a.png' },
    { id: 2, contentType: 'application/pdf', fileName: 'b.pdf' },
    { id: 3, contentType: '', fileName: 'data.json' },
  ]
  it('returns all when category is empty', () => {
    expect(filterByCategory(items, '')).toHaveLength(3)
  })
  it('filters by category key', () => {
    expect(filterByCategory(items, 'image').map((i) => i.id)).toEqual([1])
    expect(filterByCategory(items, 'pdf').map((i) => i.id)).toEqual([2])
    expect(filterByCategory(items, 'code').map((i) => i.id)).toEqual([3])
    expect(filterByCategory(items, 'archive')).toEqual([])
  })
})

describe('countByCategory', () => {
  it('tallies each category in one pass', () => {
    const items = [
      { contentType: 'image/png', fileName: 'a.png' },
      { contentType: 'image/jpeg', fileName: 'b.jpg' },
      { contentType: 'application/pdf', fileName: 'c.pdf' },
      { contentType: '', fileName: 'd.sql' },
      { contentType: '', fileName: 'e.unknown' },
    ]
    expect(countByCategory(items)).toEqual({
      image: 2, audio: 0, video: 0, pdf: 1, document: 0,
      spreadsheet: 0, archive: 0, code: 1, other: 1,
    })
  })
  it('returns all zeros for empty input', () => {
    expect(countByCategory([])).toEqual({
      image: 0, audio: 0, video: 0, pdf: 0, document: 0,
      spreadsheet: 0, archive: 0, code: 0, other: 0,
    })
  })
})