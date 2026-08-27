// 师傅详情抽屉 i18n 引用键门禁:worker-detail-view.ts 规格引用的全部文案键
// (列名 k / 枚举 map 值 / 主档字段 labelKey / 段名 / 状态徽 WORKER_STATUS_KEY /
// 抽屉字面量 DRAWER_W_KEYS)必须存在于三份 locale 的 workerPage.{d,sectionNames} 与顶层键。
// workerPage.d 是宽松 Record<string,string>,typecheck 兜不住打错的引用键,
// 一旦打错会在线上直接露出英文键值——本测试把失配变成门禁红灯。
// 新增段/列/枚举/字段时自动纳入 walker;抽屉字面量须同步 DRAWER_W_KEYS。
import { describe, expect, it } from 'vitest'
import zhCN from '../../../i18n/locales/zh-CN'
import enUS from '../../../i18n/locales/en-US'
import msMY from '../../../i18n/locales/ms-MY'
import {
  DRAWER_W_KEYS, WORKER_DETAIL_SECTIONS, WORKER_PROFILE_FIELDS, WORKER_STATUS_KEY,
} from './worker-detail-view'
import { REGISTRY } from '../../../components/StatusTag/registry'

interface RefKeys {
  dKeys: Set<string>
  topKeys: Set<string>
  sections: Set<string>
}

function referencedKeys(): RefKeys {
  const dKeys = new Set<string>()
  const topKeys = new Set<string>()
  const sections = new Set<string>()
  for (const s of WORKER_DETAIL_SECTIONS) {
    sections.add(s.key)
    for (const col of s.cols) {
      dKeys.add(col.k)
      if (col.spec?.kind === 'enum') {
        for (const v of Object.values(col.spec.map)) dKeys.add(v)
      }
    }
  }
  for (const f of WORKER_PROFILE_FIELDS) dKeys.add(f.labelKey)
  for (const v of Object.values(WORKER_STATUS_KEY)) topKeys.add(v) // active/left 是 workerPage 顶层键
  for (const k of DRAWER_W_KEYS) dKeys.add(k)
  return { dKeys, topKeys, sections }
}

describe('师傅详情抽屉 i18n 引用键门禁', () => {
  const ref = referencedKeys()

  it.each([
    ['zh-CN', zhCN],
    ['en-US', enUS],
    ['ms-MY', msMY],
  ] as const)('%s 字典覆盖 worker-detail 全部引用键', (_name, locale) => {
    const w = locale.pages.workerPage
    const missingD = [...ref.dKeys].filter((k) => !(k in w.d))
    expect(missingD, '缺 workerPage.d 字典键(引用键打错或三语言字典未同步)').toEqual([])
    const missingT = [...ref.topKeys].filter((k) => !(k in w))
    expect(missingT, '缺 workerPage 顶层键(active/left 状态徽)').toEqual([])
    const missingS = [...ref.sections].filter((k) => !(k in w.sectionNames))
    expect(missingS, '缺 workerPage.sectionNames 段名键').toEqual([])
  })

  it('关联子集段 limit 为明确正整数(超限即折叠)', () => {
    for (const s of WORKER_DETAIL_SECTIONS) {
      expect(Number.isInteger(s.limit) && s.limit > 0, `${s.key} 段上限应为正整数`).toBe(true)
    }
  })

  it('WorkTypeRaw 枚举引用完整对齐 StatusTag 注册域(ticket/message)', () => {
    // ticket/message 状态渲染走 StatusTag:注册域必须存在对应值,缺失即渲染 unknown 灰标。
    // StatusTag.test 已双向锁定 registry↔locale,这里仅断言本规格引用的域已注册。
    for (const s of WORKER_DETAIL_SECTIONS) {
      for (const col of s.cols) {
        if (col.spec?.kind === 'tag') {
          expect(REGISTRY[col.spec.domain as keyof typeof REGISTRY], `StatusTag 域 ${col.spec.domain} 未注册`).toBeDefined()
        }
      }
    }
  })
})