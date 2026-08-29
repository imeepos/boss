// 菜单权限三方对账门禁(menu.def × ROLE_GROUPS × 线上 menu-perms 快照)。
// 基线=采集日已接受漂移;断言"只减不增":新增 FE有BE无(可见即403面)或
// BE有FE无(授权被藏/前端缺页)即红。修复漂移后:
//   1) node scripts/fetch-menu-perms.mjs  重新采集线上矩阵
//   2) MENU_REGEN=1 pnpm vitest run src/router/menu-sync.test.ts  刷新基线
import { describe, expect, it } from 'vitest'
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { MENU_GROUPS, PAGE_BY_KEY } from './menu.def'
import { ROLE_GROUPS } from './role-menu'

const here = dirname(fileURLToPath(import.meta.url))
const read = (f: string) => JSON.parse(readFileSync(resolve(here, '__fixtures__', f), 'utf8'))
const snapshot = read('menu-perms-snapshot.json')

/** FE 页集:内置角色 = ROLE_GROUPS 组级展开;未知角色/自定义 = 权限码∩menu.def 真实页面集(侧栏权限码推导语义)。 */
function fePagesForRole(role: string, beCodes: string[]): Set<string> {
  const groups = (ROLE_GROUPS as Record<string, string[]>)[role]
  if (!groups) return new Set(beCodes.map((c) => c.replace(/^menu:/, '')).filter((k) => PAGE_BY_KEY.has(k)))
  const ids = new Set(groups)
  return new Set(MENU_GROUPS.filter((g) => ids.has(g.id)).flatMap((g) => g.items.map((i) => i.key)))
}

function currentDrift(): Record<string, { feOnly: string[]; beOnly: string[] }> {
  const drift: Record<string, { feOnly: string[]; beOnly: string[] }> = {}
  for (const role of Object.keys(snapshot.roles)) {
    const be = new Set<string>(snapshot.roles[role]?.be ?? [])
    const fe = fePagesForRole(role, snapshot.roles[role]?.be ?? [])
    drift[role] = {
      feOnly: [...fe].filter((k) => !be.has(`menu:${k}`)).sort(),
      beOnly: [...be].filter((c) => !fe.has(c.replace(/^menu:/, ''))).sort(),
    }
  }
  return drift
}

function subsetOf(current: string[], baseline: string[]): string[] {
  const base = new Set(baseline)
  return current.filter((k) => !base.has(k))
}

describe('menu 权限三方对账(只减不增)', () => {
  const baselineFile = resolve(here, '__fixtures__', 'menu-drift-baseline.json')

  it('FE/BE 差集不得超出基线(新漂移即红)', () => {
    const drift = currentDrift()
    if (process.env.MENU_REGEN) {
      writeFileSync(baselineFile, JSON.stringify({ note: '采集日已接受漂移;门禁=只减不增;修复后重跑 fetch-menu-perms.mjs + MENU_REGEN=1 刷新', capturedAt: snapshot.capturedAt, drift }, null, 2) + '\n')
      console.log('baseline regenerated at', baselineFile)
      return
    }
    let baseline: { drift: Record<string, { feOnly: string[]; beOnly: string[] }> }
    try {
      baseline = read('menu-drift-baseline.json')
    } catch {
      throw new Error('基线缺失:先 MENU_REGEN=1 pnpm vitest run src/router/menu-sync.test.ts 生成')
    }
    const regressions: string[] = []
    for (const role of Object.keys(drift)) {
      const base = baseline.drift[role] ?? { feOnly: [], beOnly: [] }
      for (const k of subsetOf(drift[role].feOnly, base.feOnly)) regressions.push(`${role} FE有BE无 +menu:${k}`)
      for (const k of subsetOf(drift[role].beOnly, base.beOnly)) regressions.push(`${role} BE有FE无 ${role === k ? '' : 'menu:'}${k}`)
    }
    expect(regressions, `发现新增菜单权限漂移(修复或刷新基线):\n${regressions.join('\n')}`).toEqual([])
  })
})
