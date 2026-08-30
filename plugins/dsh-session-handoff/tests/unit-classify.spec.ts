import { describe, expect, it } from 'vitest'
import { DEFAULT_CONFIG } from '../src/invariant.js'
import { classify } from '../src/survey.js'
import { dirty, facts } from './helpers.js'

interface Row {
  name: string
  overrides?: Partial<ReturnType<typeof facts>>
  mainBranch?: string
  remote?: string
  expect: Array<{ priority: string; titleIncludes: string }>
}

const TABLE: Row[] = [
  {
    name: '干净项目零任务',
    expect: [],
  },
  {
    name: '主分支脏改升级为 P0',
    overrides: { branch: 'main', dirtyFiles: dirty('a.ts', 'b.ts') },
    expect: [{ priority: 'P0', titleIncludes: '主分支 main 上有 2 个未提交改动' }],
  },
  {
    name: '普通分支脏改是 P1',
    overrides: { dirtyFiles: dirty('a.ts') },
    expect: [{ priority: 'P1', titleIncludes: '1 个未提交文件' }],
  },
  {
    name: '有上游且领先是 P1 推送',
    overrides: { ahead: 2, unpushed: ['x1 feat: a', 'x2 fix: b'] },
    expect: [{ priority: 'P1', titleIncludes: '2 个 commit 未推送到 gitea' }],
  },
  {
    name: '无上游的功能分支是 P1 建上游',
    overrides: { hasUpstream: false },
    expect: [{ priority: 'P1', titleIncludes: '没有上游' }],
  },
  {
    name: '无上游的主分支不建任务（主分支可以无上游）',
    overrides: { hasUpstream: false, branch: 'main' },
    expect: [],
  },
  {
    name: '落后远端是 P2 反向同步',
    overrides: { behind: 4 },
    expect: [{ priority: 'P2', titleIncludes: '落后远端 4 个提交' }],
  },
  {
    name: '残留 worktree 是 P2',
    overrides: { otherWorktrees: ['/wt-a (feat/a)'] },
    expect: [{ priority: 'P2', titleIncludes: '1 个 worktree 残留' }],
  },
  {
    name: '未合并分支是 P2',
    overrides: { unmergedBranches: ['feat/old', 'fix/dead'] },
    expect: [{ priority: 'P2', titleIncludes: '2 个分支未合入 main' }],
  },
  {
    name: '未合并清单过滤主分支与当前分支',
    overrides: { branch: 'feat/cur', unmergedBranches: ['main', 'feat/cur', 'feat/keep'] },
    expect: [{ priority: 'P2', titleIncludes: '1 个分支未合入 main' }],
  },
  {
    name: 'stash 是 P3',
    overrides: { stashCount: 3 },
    expect: [{ priority: 'P3', titleIncludes: '3 条暂存记录' }],
  },
  {
    name: '自定义主分支名参与判定',
    overrides: { branch: 'trunk', dirtyFiles: dirty('a.ts') },
    mainBranch: 'trunk',
    expect: [{ priority: 'P0', titleIncludes: '主分支 trunk' }],
  },
  {
    name: '自定义远端名进话术',
    overrides: { ahead: 1 },
    remote: 'origin',
    expect: [{ priority: 'P1', titleIncludes: '未推送到 origin' }],
  },
  {
    name: '多信号全触发并按优先级排序',
    overrides: {
      branch: 'main',
      dirtyFiles: dirty('a.ts'),
      ahead: 1,
      behind: 1,
      otherWorktrees: ['/wt (b)'],
      unmergedBranches: ['old'],
      stashCount: 1,
    },
    expect: [
      { priority: 'P0', titleIncludes: '主分支' },
      { priority: 'P1', titleIncludes: '未推送到' },
      { priority: 'P2', titleIncludes: '落后远端' },
      { priority: 'P2', titleIncludes: 'worktree 残留' },
      { priority: 'P2', titleIncludes: '未合入 main' },
      { priority: 'P3', titleIncludes: 'stash' },
    ],
  },
]

describe('classify 规则表', () => {
  for (const row of TABLE) {
    it(row.name, () => {
      const config = { ...DEFAULT_CONFIG, ...(row.mainBranch ? { mainBranch: row.mainBranch } : {}), ...(row.remote ? { remote: row.remote } : {}) }
      const tasks = classify(facts(row.overrides), config)
      expect(tasks.map((t) => ({ priority: t.priority, title: t.title })).filter((t) => row.expect.some((e) => t.priority === e.priority && t.title.includes(e.titleIncludes))))
        .toHaveLength(row.expect.length)
      expect(tasks).toHaveLength(row.expect.length)
    })
  }
})

describe('classify 细节', () => {
  it('P0 主分支任务的动作为挪分支或清理', () => {
    const tasks = classify(facts({ branch: 'main', dirtyFiles: dirty('a.ts') }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('worktree')
  })

  it('P1 脏改任务带提交话术', () => {
    const tasks = classify(facts({ dirtyFiles: dirty('a.ts') }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('git commit')
  })

  it('推送任务带上游分支名', () => {
    const tasks = classify(facts({ branch: 'feat/x', ahead: 1 }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('git push gitea feat/x')
  })

  it('建上游任务带 -u 参数', () => {
    const tasks = classify(facts({ hasUpstream: false, branch: 'feat/new' }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('git push -u gitea feat/new')
  })

  it('反向同步任务指向远端主分支', () => {
    const tasks = classify(facts({ behind: 1 }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('git merge gitea/main')
  })

  it('worktree 任务提示收尾四步', () => {
    const tasks = classify(facts({ otherWorktrees: ['/wt (b)'] }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('收尾四步')
  })

  it('未合并任务给逐个确认建议', () => {
    const tasks = classify(facts({ unmergedBranches: ['old'] }), DEFAULT_CONFIG)
    expect(tasks[0].actions.join(' ')).toContain('合并回主分支')
  })

  it('脏改预览最多列 5 个文件并标注总数', () => {
    const many = dirty('1.ts', '2.ts', '3.ts', '4.ts', '5.ts', '6.ts', '7.ts')
    const tasks = classify(facts({ dirtyFiles: many }), DEFAULT_CONFIG)
    expect(tasks[0].detail).toContain('等 7 项')
    expect(tasks[0].detail).not.toContain('6.ts')
  })

  it('未推送预览最多 3 条并标注总数', () => {
    const tasks = classify(facts({ ahead: 4, unpushed: ['a', 'b', 'c', 'd'] }), DEFAULT_CONFIG)
    expect(tasks[0].detail).toBe('a、b、c 等 4 项')
  })

  it('排序稳定：P0 在 P1 在 P2 在 P3 前', () => {
    const tasks = classify(
      facts({ dirtyFiles: dirty('a.ts'), behind: 1, stashCount: 1, otherWorktrees: ['/wt (b)'] }),
      DEFAULT_CONFIG,
    )
    expect(tasks.map((t) => t.priority)).toEqual(['P1', 'P2', 'P2', 'P3'])
  })
})

describe('classify 边界组合', () => {
  it('主分支脏改超过 5 个文件时预览标注总数', () => {
    const many = dirty('1.ts', '2.ts', '3.ts', '4.ts', '5.ts', '6.ts')
    const tasks = classify(facts({ branch: 'main', dirtyFiles: many }), DEFAULT_CONFIG)
    expect(tasks).toHaveLength(1)
    expect(tasks[0].detail).toContain('等 6 项')
  })

  it('无上游的主分支 + 脏改只出 P0 一条', () => {
    const tasks = classify(facts({ branch: 'main', hasUpstream: false, dirtyFiles: dirty('a.ts') }), DEFAULT_CONFIG)
    expect(tasks.map((t) => t.priority)).toEqual(['P0'])
  })

  it('主分支领先远端也照常建议推送', () => {
    const tasks = classify(facts({ branch: 'main', ahead: 2 }), DEFAULT_CONFIG)
    expect(tasks).toHaveLength(1)
    expect(tasks[0].title).toContain('2 个 commit 未推送到 gitea')
  })

  it('未合并超过 5 个时截断并标注总数', () => {
    const branches = ['b1', 'b2', 'b3', 'b4', 'b5', 'b6', 'b7']
    const tasks = classify(facts({ unmergedBranches: branches }), DEFAULT_CONFIG)
    expect(tasks[0].detail).toContain('b1')
    expect(tasks[0].detail).toContain('b5')
    expect(tasks[0].detail).not.toContain('b6')
    expect(tasks[0].detail).toContain('等 7 项')
  })

  it('worktree 恰好 5 个不截断', () => {
    const wts = ['/w1 (a)', '/w2 (b)', '/w3 (c)', '/w4 (d)', '/w5 (e)']
    const tasks = classify(facts({ otherWorktrees: wts }), DEFAULT_CONFIG)
    expect(tasks[0].detail).toBe(wts.join('、'))
  })

  it('同输入两次分类结果一致（纯函数无漂移）', () => {
    const input = facts({ dirtyFiles: dirty('a.ts'), ahead: 1, stashCount: 2 })
    expect(classify(input, DEFAULT_CONFIG)).toEqual(classify(input, DEFAULT_CONFIG))
  })

  it('unmerged 只含当前分支与主分支时不产任务', () => {
    const tasks = classify(facts({ branch: 'dev', unmergedBranches: ['main', 'dev'] }), DEFAULT_CONFIG)
    expect(tasks).toHaveLength(0)
  })

  it('P0 与 P1 脏改互斥：主分支不重复计任务', () => {
    const tasks = classify(facts({ branch: 'main', dirtyFiles: dirty('a.ts', 'b.ts') }), DEFAULT_CONFIG)
    expect(tasks.filter((t) => t.title.includes('未提交'))).toHaveLength(1)
  })

  it('领先与落后同时存在出两条任务', () => {
    const tasks = classify(facts({ ahead: 1, behind: 1 }), DEFAULT_CONFIG)
    expect(tasks.map((t) => t.priority)).toEqual(['P1', 'P2'])
  })

  it('只有 stash 时恰出一条 P3', () => {
    const tasks = classify(facts({ stashCount: 1 }), DEFAULT_CONFIG)
    expect(tasks.map((t) => `${t.priority} ${t.title}`)).toEqual(['P3 stash 里有 1 条暂存记录'])
  })
})
