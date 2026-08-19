import { describe, it, expect } from 'vitest'
import { buildCreatePayload } from './payload'

describe('buildCreatePayload', () => {
  it('账号主体契约:subjectType=account + subjectRef=accountId', () => {
    expect(buildCreatePayload(103, 'ci-key')).toEqual({
      subjectType: 'account', subjectRef: 103, name: 'ci-key',
    })
  })

  it('name 去空白;回归:禁止回到旧 {accountId,name} 形状', () => {
    const p = buildCreatePayload(1, '  padded  ') as Record<string, unknown>
    expect(p.name).toBe('padded')
    expect('accountId' in p).toBe(false)
  })
})
