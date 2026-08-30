import type { GitFacts } from '../src/types.js'

/** 事实夹具：干净分支起步，测试按需覆盖字段。 */
export function facts(overrides: Partial<GitFacts> = {}): GitFacts {
  return {
    toplevel: '/repo',
    branch: 'feat/x',
    dirtyFiles: [],
    lastCommit: 'abc1234 feat: base',
    hasUpstream: true,
    ahead: 0,
    behind: 0,
    unpushed: [],
    otherWorktrees: [],
    unmergedBranches: [],
    stashCount: 0,
    ...overrides,
  }
}

/** dirty 文件简写。 */
export function dirty(...paths: string[]): GitFacts['dirtyFiles'] {
  return paths.map((path) => ({ code: ' M', path }))
}
