import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { StatusTag, statusTagLabel } from './index'
import { REGISTRY } from './registry'
import { LocaleProvider } from '../../i18n/context'
import zhCN from '../../i18n/locales/zh-CN'
import enUS from '../../i18n/locales/en-US'
import msMY from '../../i18n/locales/ms-MY'

// 契约:① 全部枚举注册且渲染不出 unknown;颜色语义集中定义一次;
// ② registry 域×值 与三语 common.statusTags 键集双向相等——新状态忘补翻译、
// 状态下线字典残键,双双 CI 红灯;③ 三语言各自真实渲染对应标签。

function registryKeys(): string[] {
  return Object.entries(REGISTRY)
    .flatMap(([d, vals]) => Object.keys(vals as Record<string, string>).map((v) => `${d}.${v}`))
    .sort()
}

describe('StatusTag 注册表与三语字典完备性', () => {
  it('覆盖 24 个枚举域(与 server-ts/src/enums.ts 状态类对齐)', () => {
    expect(Object.keys(REGISTRY).length).toBeGreaterThanOrEqual(24)
  })

  it('每域非空且每值为色名', () => {
    for (const [, vals] of Object.entries(REGISTRY)) {
      expect(Object.keys(vals as Record<string, string>).length).toBeGreaterThan(0)
      for (const color of Object.values(vals as Record<string, string>)) {
        expect(color).toMatch(/^#[0-9a-f]{6}$/)
      }
    }
  })

  for (const [name, dict] of [['zh-CN', zhCN], ['en-US', enUS], ['ms-MY', msMY]] as const) {
    it(`${name}:registry 全键都有翻译(新增状态必须三语同步)`, () => {
      const tags = Object.keys(dict.common.statusTags)
      expect(registryKeys().filter((k) => !tags.includes(k))).toEqual([])
    })
    it(`${name}:statusTags 无残键(状态下线字典同步清理)`, () => {
      expect(Object.keys(dict.common.statusTags).filter((k) => !registryKeys().includes(k))).toEqual([])
    })
    it(`${name}:标签非空`, () => {
      for (const v of Object.values(dict.common.statusTags)) expect(v.length).toBeGreaterThan(0)
    })
  }
})

describe('StatusTag 渲染', () => {
  it('zh-CN/en-US/ms-MY 各自渲染对应语言标签', () => {
    const html = (locale: 'zh-CN' | 'en-US' | 'ms-MY') =>
      renderToStaticMarkup(
        <LocaleProvider localeOverride={locale}>
          <StatusTag domain="order" value="INSTALLING" />
        </LocaleProvider>,
      )
    expect(html('zh-CN')).toContain('装维中')
    expect(html('en-US')).toContain('Installing')
    expect(html('ms-MY')).toContain('Pemasangan')
  })

  it('statusTagLabel 缺键回退枚举原文;未注册域/值直返原文', () => {
    expect(statusTagLabel('order', 'INSTALLING', {})).toBe('INSTALLING')
    expect(statusTagLabel('nope', 'X', zhCN.common.statusTags)).toBe('X')
  })

  it('未注册值渲染 unknown(灰)标记,不抛错', () => {
    const html = (d: string, v: string) =>
      renderToStaticMarkup(
        <LocaleProvider localeOverride="zh-CN">
          <StatusTag domain={d} value={v} />
        </LocaleProvider>,
      )
    expect(html('order', '???')).toContain('st-unknown')
    expect(html('order', '???')).toContain('???')
  })

  it('快照:全枚举渲染无遗漏(出现 unknown 标记即失败)', () => {
    let count = 0
    for (const [domain, vals] of Object.entries(REGISTRY)) {
      for (const value of Object.keys(vals as Record<string, string>)) {
        const html = renderToStaticMarkup(
          <LocaleProvider localeOverride="zh-CN">
            <StatusTag domain={domain} value={value} />
          </LocaleProvider>,
        )
        expect(html).not.toContain('st-unknown')
        count++
      }
    }
    expect(count).toBeGreaterThanOrEqual(80)
  })
})
