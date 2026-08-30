import { describe, expect, it } from 'vitest'
import { SURVEY_SCRIPT } from '../src/survey.js'

describe('SURVEY_SCRIPT 契约（插件与 fake shell 共同依赖的输出协议）', () => {
  it('每个分段标记恰好出现一次', () => {
    for (const key of ['toplevel', 'branch', 'status', 'lastcommit', 'aheadbehind', 'unpushed', 'worktrees', 'unmerged', 'stash', 'end']) {
      const hits = SURVEY_SCRIPT.split('\n').filter((l) => l.includes(`###MARK ${key}`))
      expect(hits, `marker ${key}`).toHaveLength(1)
    }
  })

  it('toplevel 用 rev-parse --show-toplevel', () => {
    expect(SURVEY_SCRIPT).toContain('git rev-parse --show-toplevel')
  })

  it('branch 用 show-current', () => {
    expect(SURVEY_SCRIPT).toContain('git branch --show-current')
  })

  it('status 用 porcelain 格式', () => {
    expect(SURVEY_SCRIPT).toContain('git status --porcelain')
  })

  it('lastcommit 取 hash 与 subject', () => {
    const line = SURVEY_SCRIPT.split('\n').find((l) => l.includes('git log -1'))!
    expect(line).toContain('%h')
    expect(line).toContain('%s')
  })

  it('aheadbehind 用 left-right count 且上游引用带引号', () => {
    const line = SURVEY_SCRIPT.split('\n').find((l) => l.includes('rev-list'))!
    expect(line).toContain('--left-right --count')
    expect(line).toContain('"@{upstream}...HEAD"')
  })

  it('unpushed 限 10 条', () => {
    const line = SURVEY_SCRIPT.split('\n').find((l) => l.includes('"@{upstream}..HEAD"'))!
    expect(line).toContain('-n 10')
    expect(line).toContain('--oneline')
  })

  it('worktrees 用 porcelain', () => {
    expect(SURVEY_SCRIPT).toContain('git worktree list --porcelain')
  })

  it('unmerged 相对 HEAD', () => {
    expect(SURVEY_SCRIPT).toContain('git branch --no-merged')
  })

  it('stash 只数条数', () => {
    expect(SURVEY_SCRIPT).toContain('git stash list')
  })

  it('set +e 保证单段失败不中断后续分段', () => {
    expect(SURVEY_SCRIPT.split('\n')[0]).toBe('set +e')
  })

  it('每条 git 命令独占一行，便于 fake 按行核对', () => {
    const gitLines = SURVEY_SCRIPT.split('\n').filter((l) => l.includes('git '))
    expect(gitLines.length).toBeGreaterThanOrEqual(9)
    for (const line of gitLines) {
      expect(line.split(';').filter((p) => p.includes('git '))).toHaveLength(1)
    }
  })

  it('脚本无 rm/mv/push 等破坏性命令（盘点只读）', () => {
    for (const dangerous of ['rm ', 'mv ', 'git push', 'git reset', 'git clean', 'git checkout', 'git merge', 'git rebase', 'git stash drop']) {
      expect(SURVEY_SCRIPT).not.toContain(dangerous)
    }
  })
})
