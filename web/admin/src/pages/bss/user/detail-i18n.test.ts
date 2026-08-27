// 用户详情抽屉 i18n 引用键门禁:detail-view.ts 规格引用的全部文案键
// (列名 k / 枚举 map 值 / 段名 / 页签名 / 主档 d_* / 抽屉字面量 DRAWER_D_KEYS)
// 必须存在于三份 locale 的 pages.userPage.{d,sectionNames,detailTabs}。
// userPage.d 是宽松 Record<string,string>,typecheck 兜不住打错的引用键,
// 一旦打错会在线上直接露出英文键值——本测试把失配变成门禁红灯。
// 新增列/枚举/段/页签时自动纳入 walker;抽屉字面量须同步 DRAWER_D_KEYS。
import { describe, expect, it } from 'vitest'
import zhCN from '../../../i18n/locales/zh-CN'
import enUS from '../../../i18n/locales/en-US'
import msMY from '../../../i18n/locales/ms-MY'
import { DETAIL_TABS, DRAWER_D_KEYS, NOTIFY_CARD_KEY, PROFILE_FIELDS, SECTION_LIMIT } from './detail-view'

interface RefKeys { dKeys: Set<string>; sections: Set<string>; tabs: Set<string> }

function referencedKeys(): RefKeys {
  const dKeys = new Set<string>()
  const sections = new Set<string>()
  const tabs = new Set<string>()
  for (const tab of DETAIL_TABS) {
    tabs.add(tab.key)
    for (const s of tab.sections) {
      if (s.key !== NOTIFY_CARD_KEY) sections.add(s.key)
      for (const col of s.cols) {
        dKeys.add(col.k)
        if (col.spec?.kind === 'enum') {
          for (const v of Object.values(col.spec.map)) dKeys.add(v)
        }
      }
    }
  }
  for (const f of PROFILE_FIELDS) dKeys.add(`d_${f}`)
  for (const k of DRAWER_D_KEYS) dKeys.add(k)
  return { dKeys, sections, tabs }
}

describe('用户详情抽屉 i18n 引用键门禁', () => {
  const ref = referencedKeys()

  it.each([
    ['zh-CN', zhCN],
    ['en-US', enUS],
    ['ms-MY', msMY],
  ] as const)('%s 字典覆盖 detail-view 全部引用键', (_name, locale) => {
    const d = locale.pages.userPage.d
    const missingD = [...ref.dKeys].filter((k) => !(k in d))
    expect(missingD, '缺 d 字典键(引用键打错或三语言字典未同步)').toEqual([])
    const missingS = [...ref.sections].filter((k) => !(k in locale.pages.userPage.sectionNames))
    expect(missingS, '缺 sectionNames 段名键').toEqual([])
    const missingT = [...ref.tabs].filter((k) => !(k in locale.pages.userPage.detailTabs))
    expect(missingT, '缺 detailTabs 页签键').toEqual([])
  })

  it('每段列表上限为明确正整数(缺省走 SECTION_LIMIT,超限即折叠)', () => {
    for (const tab of DETAIL_TABS) {
      for (const s of tab.sections) {
        if (s.key === NOTIFY_CARD_KEY) continue
        const limit = s.limit ?? SECTION_LIMIT
        expect(Number.isInteger(limit) && limit > 0, `${s.key} 段上限应为正整数`).toBe(true)
      }
    }
  })
})
