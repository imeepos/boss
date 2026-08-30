import { describe, expect, it } from 'vitest'
import { parseAheadBehind, parseFacts, parseStatusLine, parseWorktrees, splitSurvey, SURVEY_SCRIPT } from '../src/survey.js'

/** 把分段表组装成 survey 脚本样式的原始输出。 */
function survey(sections: Record<string, string>): string {
  const parts = Object.entries(sections).map(([key, body]) => `###MARK ${key}\n${body}`)
  return `${parts.join('\n')}\n###MARK end`
}

const FULL = survey({
  toplevel: '/repoalpha',
  branch: 'feat/demo',
  status: ' M src/a.ts\n?? notes.md\nR  old.ts -> new.ts',
  lastcommit: 'abc1234 feat(demo): x',
  aheadbehind: '2\t3',
  unpushed: 'abc1234 feat(demo): x\ndef5678 fix(demo): y',
  worktrees: 'worktree /repoalpha\nHEAD abc1234\nbranch refs/heads/feat/demo\n\nworktree /wt-other\nHEAD 1111111\nbranch refs/heads/fix/other\n',
  unmerged: '  fix/other\n* feat/demo',
  stash: 'stash@{0}: WIP on feat/demo: abc1234 x\nstash@{1}: On main: wip',
})

describe('splitSurvey 分段', () => {
  it('完整输出按标记切段', () => {
    const s = splitSurvey(FULL)
    expect(Object.keys(s).sort()).toEqual(['aheadbehind', 'branch', 'end', 'lastcommit', 'stash', 'status', 'toplevel', 'unmerged', 'unpushed', 'worktrees'])
  })

  it('首标记前的内容被丢弃', () => {
    const s = splitSurvey('noise before\n###MARK a\n1\n')
    expect(s.a).toBe('1\n')
  })

  it('段内容保留换行结构', () => {
    const s = splitSurvey('###MARK a\nl1\nl2\n###MARK b\nx')
    expect(s.a).toBe('l1\nl2')
    expect(s.b).toBe('x')
  })

  it('缺尾部标记也不丢段', () => {
    const s = splitSurvey('###MARK only\nvalue')
    expect(s.only).toBe('value')
  })

  it('空输入返回空表', () => {
    expect(splitSurvey('')).toEqual({})
  })

  it('连续标记产生空段', () => {
    const s = splitSurvey('###MARK a\n###MARK b\nv')
    expect(s.a).toBe('')
    expect(s.b).toBe('v')
  })

  it('SURVEY_SCRIPT 含全部分段标记', () => {
    for (const key of ['toplevel', 'branch', 'status', 'lastcommit', 'aheadbehind', 'unpushed', 'worktrees', 'unmerged', 'stash']) {
      expect(SURVEY_SCRIPT).toContain(`###MARK ${key}`)
    }
  })

  it('SURVEY_SCRIPT 以 set +e 开头保证不中断', () => {
    expect(SURVEY_SCRIPT.startsWith('set +e')).toBe(true)
  })
})

describe('parseFacts 全量解析', () => {
  const f = parseFacts(FULL)

  it('项目根与分支', () => {
    expect(f.toplevel).toBe('/repoalpha')
    expect(f.branch).toBe('feat/demo')
  })

  it('未提交文件含重命名取新路径', () => {
    expect(f.dirtyFiles.map((d) => d.path)).toEqual(['src/a.ts', 'notes.md', 'new.ts'])
    expect(f.dirtyFiles.map((d) => d.code)).toEqual([' M', '??', 'R '])
  })

  it('ahead/behind 计数与未推列表', () => {
    expect(f.ahead).toBe(3)
    expect(f.behind).toBe(2)
    expect(f.unpushed).toHaveLength(2)
  })

  it('主树之外的 worktree 被挑出并带分支', () => {
    expect(f.otherWorktrees).toEqual(['/wt-other (fix/other)'])
  })

  it('未合并分支剥掉修饰符', () => {
    expect(f.unmergedBranches).toEqual(['fix/other', 'feat/demo'])
  })

  it('stash 计数', () => {
    expect(f.stashCount).toBe(2)
  })

  it('最近提交原样保留', () => {
    expect(f.lastCommit).toBe('abc1234 feat(demo): x')
  })
})

describe('parseFacts 空态与降级', () => {
  const cases: Array<{ name: string; raw: string; expect: Record<string, unknown> }> = [
    { name: '空输出', raw: '', expect: { toplevel: '', branch: '', hasUpstream: false, ahead: 0, behind: 0 } },
    { name: '非 git 目录（全段空）', raw: survey({ toplevel: '', branch: '' }), expect: { toplevel: '', branch: '', hasUpstream: false } },
    { name: '有上游但计数为 0', raw: survey({ aheadbehind: '0\t0' }), expect: { hasUpstream: true, ahead: 0, behind: 0 } },
    { name: '计数输出损坏时保守归零', raw: survey({ aheadbehind: 'garbage' }), expect: { hasUpstream: true, ahead: 0, behind: 0 } },
    { name: '缺段不炸', raw: '###MARK branch\nmain\n', expect: { branch: 'main', toplevel: '' } },
    { name: 'detached HEAD 分支为空串', raw: survey({ branch: '' }), expect: { branch: '' } },
    { name: 'status 空行被忽略', raw: survey({ status: '\n\n' }), expect: { dirtyFiles: [] } },
    { name: 'unmerged 空白剥净', raw: survey({ unmerged: '  \n' }), expect: { unmergedBranches: [] } },
  ]
  for (const c of cases) {
    it(c.name, () => {
      expect(parseFacts(c.raw)).toMatchObject(c.expect)
    })
  }
})

describe('parseAheadBehind', () => {
  const cases: Array<{ name: string; input: string; expect: { ahead: number; behind: number; hasUpstream: boolean } }> = [
    { name: 'behind 在前 ahead 在后', input: '2\t3', expect: { behind: 2, ahead: 3, hasUpstream: true } },
    { name: '空输出无上游', input: '', expect: { behind: 0, ahead: 0, hasUpstream: false } },
    { name: '空白输出无上游', input: '  \n', expect: { behind: 0, ahead: 0, hasUpstream: false } },
    { name: '单列损坏保守归零但视为有上游', input: '7', expect: { behind: 0, ahead: 0, hasUpstream: true } },
    { name: '非数字损坏同理', input: 'a\tb', expect: { behind: 0, ahead: 0, hasUpstream: true } },
    { name: '空格分隔也认', input: '1 2', expect: { behind: 1, ahead: 2, hasUpstream: true } },
  ]
  for (const c of cases) {
    it(c.name, () => {
      expect(parseAheadBehind(c.input)).toEqual(c.expect)
    })
  }
})

describe('parseStatusLine', () => {
  const cases: Array<{ name: string; input: string; expect: unknown }> = [
    { name: '工作区修改', input: ' M a.ts', expect: { code: ' M', path: 'a.ts' } },
    { name: '未跟踪', input: '?? b.ts', expect: { code: '??', path: 'b.ts' } },
    { name: '重命名取新路径', input: 'R  old -> new', expect: { code: 'R ', path: 'new' } },
    { name: '带引号路径去引号', input: '?? "a b.ts"', expect: { code: '??', path: 'a b.ts' } },
    { name: '过短行丢弃', input: '??', expect: null },
    { name: '空行丢弃', input: '', expect: null },
    { name: '双列修改', input: 'MM x.ts', expect: { code: 'MM', path: 'x.ts' } },
    { name: '删除标记', input: ' D gone.ts', expect: { code: ' D', path: 'gone.ts' } },
  ]
  for (const c of cases) {
    it(c.name, () => {
      expect(parseStatusLine(c.input)).toEqual(c.expect)
    })
  }
})

describe('parseWorktrees', () => {
  const LIST = [
    'worktree /main-tree',
    'HEAD abc1234',
    'branch refs/heads/main',
    '',
    'worktree /wt-a',
    'HEAD 1111111',
    'branch refs/heads/feat/a',
    '',
    'worktree /wt-b',
    'HEAD 2222222',
    'branch refs/heads/fix/b',
    '',
  ].join('\n')

  it('剔除主树并列出其余', () => {
    expect(parseWorktrees(LIST, '/main-tree')).toEqual(['/wt-a (feat/a)', '/wt-b (fix/b)'])
  })

  it('无主树匹配时全部保留', () => {
    expect(parseWorktrees(LIST, '/elsewhere')).toHaveLength(3)
  })

  it('detached worktree（无 branch 行）跳过', () => {
    const detached = 'worktree /main\nHEAD abc\n\nworktree /wt\nHEAD def\n'
    expect(parseWorktrees(detached, '/main')).toEqual([])
  })

  it('空输出返回空数组', () => {
    expect(parseWorktrees('', '/main')).toEqual([])
  })

  it('三个 worktree 全部列出', () => {
    const list = [
      'worktree /m',
      'branch refs/heads/main',
      '',
      'worktree /w1',
      'branch refs/heads/a',
      '',
      'worktree /w2',
      'branch refs/heads/b',
      '',
      'worktree /w3',
      'branch refs/heads/c',
      '',
    ].join('\n')
    expect(parseWorktrees(list, '/m')).toEqual(['/w1 (a)', '/w2 (b)', '/w3 (c)'])
  })

  it('嵌套路径不会误切（worktree 前缀匹配整词）', () => {
    const list = 'worktree /repo\nbranch refs/heads/main\n\nworktree /repo-nested\nbranch refs/heads/x\n'
    expect(parseWorktrees(list, '/repo')).toEqual(['/repo-nested (x)'])
  })
})

describe('parseFacts 组合状态', () => {
  const cases: Array<{ name: string; raw: string; expect: Record<string, unknown> }> = [
    {
      name: '只有脏改没有上游',
      raw: survey({ branch: 'feat/solo', status: ' M x.ts', aheadbehind: '' }),
      expect: { dirtyFiles: [{ code: ' M', path: 'x.ts' }], hasUpstream: false, ahead: 0 },
    },
    {
      name: '只领先不落后',
      raw: survey({ aheadbehind: '0\t5', unpushed: 'a b c' }),
      expect: { behind: 0, ahead: 5, unpushed: ['a b c'] },
    },
    {
      name: '只落后不领先',
      raw: survey({ aheadbehind: '7\t0' }),
      expect: { behind: 7, ahead: 0 },
    },
    {
      name: 'unmerged 的 + 标记（别的 worktree 检出）也剥掉',
      raw: survey({ unmerged: '+ feat/elsewhere' }),
      expect: { unmergedBranches: ['feat/elsewhere'] },
    },
    {
      name: 'status 保留行首空格对位（暂存新文件 A + 空格）',
      raw: survey({ status: 'A  staged.ts' }),
      expect: { dirtyFiles: [{ code: 'A ', path: 'staged.ts' }] },
    },
    {
      name: 'status 的删除行',
      raw: survey({ status: 'D  deleted.ts' }),
      expect: { dirtyFiles: [{ code: 'D ', path: 'deleted.ts' }] },
    },
    {
      name: 'status 多行混合顺序保持',
      raw: survey({ status: '?? u1\n M m1\n?? u2' }),
      expect: { dirtyFiles: [{ code: '??', path: 'u1' }, { code: ' M', path: 'm1' }, { code: '??', path: 'u2' }] },
    },
    {
      name: 'CRLF 行尾被剥掉',
      raw: `###MARK branch\r\nfeat/crlf\r\n###MARK end\r\n`,
      expect: { branch: 'feat/crlf' },
    },
    {
      name: 'lastcommit 缺失为空串',
      raw: survey({ branch: 'main' }),
      expect: { lastCommit: '' },
    },
    {
      name: 'stash 多条计数正确',
      raw: survey({ stash: 'stash@{0}: a\nstash@{1}: b\nstash@{2}: c' }),
      expect: { stashCount: 3 },
    },
  ]
  for (const c of cases) {
    it(c.name, () => {
      expect(parseFacts(c.raw)).toMatchObject(c.expect)
    })
  }
})
