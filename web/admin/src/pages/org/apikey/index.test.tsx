// API key 管理页"绑定账号"列渲染回归:
// 后端契约 internal/domain/apikey/apikey.go 字段为 subjectType/subjectRef/subjectName,
// 列表第二列必须渲染展示名(或主体类型:#ref),不允许出现 `#undefined` 字面值。
import { describe, expect, it } from 'vitest'
import { subjectCell, type ApiKeyRow } from './index'

describe('apikey list subject column', () => {
  it('首选展示名 + 主体类型标签(account)', () => {
    expect(subjectCell({
      id: 1, subjectType: 'account', subjectRef: 103, subjectName: 'ops-main',
      name: 'ci', keyPrefix: 'boss_aaa', status: 1, lastUsedAt: '', expiresAt: '', createdAt: '',
    })).toBe('ops-main (Account)')
  })

  it('首选展示名 + 主体类型标签(worker)', () => {
    expect(subjectCell({
      id: 2, subjectType: 'worker', subjectRef: 5, subjectName: '张师傅',
      name: 'field', keyPrefix: 'boss_bbb', status: 1, lastUsedAt: '', expiresAt: '', createdAt: '',
    })).toBe('张师傅 (Worker)')
  })

  it('缺展示名时退到 `type:#ref`,绝不渲染 `#undefined`', () => {
    const out = subjectCell({
      id: 3, subjectType: 'customer', subjectRef: 100, subjectName: '',
      name: 'home', keyPrefix: 'boss_ccc', status: 1, lastUsedAt: '', expiresAt: '', createdAt: '',
    })
    expect(out).toBe('Customer:#100')
    expect(out).not.toContain('undefined')
  })

  it('缺主体类型时退到 `unknown:#ref`', () => {
    const row = {
      id: 4, subjectType: '', subjectRef: 0, subjectName: 'x',
      name: 'a', keyPrefix: 'k', status: 1, lastUsedAt: '', expiresAt: '', createdAt: '',
    } as ApiKeyRow
    const out = subjectCell(row)
    expect(out).toBe('unknown:#0')
    expect(out).not.toContain('undefined')
  })

  it('三类主体标签齐全且互斥', () => {
    for (const t of ['account', 'worker', 'customer'] as const) {
      const out = subjectCell({
        id: 9, subjectType: t, subjectRef: 1, subjectName: 'n',
        name: 'a', keyPrefix: 'k', status: 1, lastUsedAt: '', expiresAt: '', createdAt: '',
      })
      expect(out).toMatch(new RegExp(`\\((${t.charAt(0).toUpperCase()}${t.slice(1)})\\)$`))
    }
  })
})