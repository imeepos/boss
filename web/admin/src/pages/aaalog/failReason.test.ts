import { describe, expect, it } from 'vitest'
import { failReasonText } from './failReason'
import zhCN from '../../i18n/locales/zh-CN'
import enUS from '../../i18n/locales/en-US'
import msMY from '../../i18n/locales/ms-MY'

// 契约:fields.md §8A 全枚举三语均有专属文案(非回退原码);空值语义=成功行「成功」/存量失败行「不适用」。
const dicts = [['zh-CN', zhCN], ['en-US', enUS], ['ms-MY', msMY]] as const
const CODES = ['BAD_CREDENTIAL', 'LOCKED', 'NOT_FOUND', 'SUSPENDED', 'CLOSED', 'CONCURRENT_LIMIT'] as const

describe('fail_reason 枚举文案', () => {
  it('成功行空值=成功文案(三语)', () => {
    for (const [, d] of dicts) {
      expect(failReasonText({ result: 'SUCCESS' }, d.pages.aaaLogPage)).toBe(d.pages.aaaLogPage.failReasonOK)
    }
  })

  it('失败行空值(存量行)=不适用文案(三语)', () => {
    for (const [, d] of dicts) {
      expect(failReasonText({ result: 'FAILED' }, d.pages.aaaLogPage)).toBe(d.pages.aaaLogPage.failReasonNA)
    }
  })

  it.each(CODES)('%s 枚举三语均有非空专属文案(非回退原码)', (code) => {
    for (const [, d] of dicts) {
      const text = failReasonText({ result: 'FAILED', failReason: code }, d.pages.aaaLogPage)
      expect(text.length).toBeGreaterThan(0)
      expect(text).not.toBe(code)
    }
  })

  it('未知码原样透出留痕', () => {
    expect(failReasonText({ result: 'FAILED', failReason: 'FUTURE_CODE' }, zhCN.pages.aaaLogPage)).toBe('FUTURE_CODE')
  })
})
