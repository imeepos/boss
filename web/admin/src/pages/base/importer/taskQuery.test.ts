import { describe, expect, it } from 'vitest'
import { buildTaskQuery } from './taskQuery'

describe('buildTaskQuery', () => {
  it('empty filters produce no query', () => {
    expect(buildTaskQuery({ kind: '', operator: '', from: '', to: '' })).toBe('')
  })

  it('encodes kind and operator filters', () => {
    const q = buildTaskQuery({ kind: 'entity:customer', operator: '张 三', from: '', to: '' })
    expect(q).toBe('?kind=entity%3Acustomer&operator=%E5%BC%A0+%E4%B8%89')
  })

  it('uses UTC midnight for inclusive start and exclusive end', () => {
    const q = buildTaskQuery({ kind: '', operator: '', from: '2026-09-01', to: '2026-09-03' })
    expect(q).toBe('?from=2026-09-01T00%3A00%3A00.000Z&to=2026-09-03T00%3A00%3A00.000Z')
  })
})
