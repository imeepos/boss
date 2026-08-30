import { describe, expect, it } from 'vitest'
import { DEFAULT_CONFIG } from '../src/invariant.js'
import { buildReport, renderMarkdown, summarize } from '../src/survey.js'
import { dirty, facts } from './helpers.js'

const NOW = new Date('2026-08-30T08:00:00Z')

function reportOf(overrides: Parameters<typeof facts>[0] = {}) {
  return buildReport(facts(overrides), DEFAULT_CONFIG, NOW)
}

describe('buildReport', () => {
  it('项目名取目录末段', () => {
    expect(reportOf().project).toBe('repo')
    const deep = buildReport(facts({ toplevel: '/a/b/my-project' }), DEFAULT_CONFIG, NOW)
    expect(deep.project).toBe('my-project')
  })

  it('注入的时间进报告', () => {
    expect(reportOf().generatedAt).toBe('2026-08-30T08:00:00.000Z')
  })

  it('outputHint 按配置拼装', () => {
    const report = buildReport(facts(), { ...DEFAULT_CONFIG, outputDir: 'handoff', fileName: 'NOW.md' }, NOW)
    expect(report.outputHint).toBe('handoff/NOW.md')
  })

  it('内嵌 facts 与任务', () => {
    const report = reportOf({ dirtyFiles: dirty('a.ts'), stashCount: 1 })
    expect(report.facts.dirtyFiles).toHaveLength(1)
    expect(report.tasks.map((t) => t.priority)).toEqual(['P1', 'P3'])
  })

  it('空 toplevel 项目名退化为完整路径', () => {
    const report = buildReport(facts({ toplevel: '' }), DEFAULT_CONFIG, NOW)
    expect(report.project).toBe('')
  })
})

describe('renderMarkdown 结构', () => {
  it('标题带项目名', () => {
    expect(renderMarkdown(reportOf())).toContain('# 会话收尾盘点 · repo')
  })

  it('头部行含时间/分支/最近提交', () => {
    const md = renderMarkdown(reportOf({ lastCommit: 'ff0001 fix: y' }))
    expect(md).toContain('- 生成时间：2026-08-30T08:00:00.000Z')
    expect(md).toContain('- 分支：feat/x（最近提交：ff0001 fix: y）')
  })

  it('远端状态行含三个计数', () => {
    const md = renderMarkdown(reportOf({ ahead: 2, behind: 3, dirtyFiles: dirty('a.ts', 'b.ts') }))
    expect(md).toContain('- 与远端：ahead 2 / behind 3 / 未提交 2 个文件')
  })

  it('每条任务的每个动作都是复选框', () => {
    const report = reportOf({ dirtyFiles: dirty('a.ts'), stashCount: 2 })
    const md = renderMarkdown(report)
    const boxes = md.split('\n').filter((l) => l.startsWith('- [ ]'))
    expect(boxes).toHaveLength(report.tasks.reduce((n, t) => n + t.actions.length, 0))
  })

  it('任务标题以 ### 和优先级开头', () => {
    const md = renderMarkdown(reportOf({ dirtyFiles: dirty('a.ts') }))
    expect(md).toContain('### P1 分支 feat/x 有 1 个未提交文件')
  })

  it('detail 行渲染在标题之后', () => {
    const md = renderMarkdown(reportOf({ dirtyFiles: dirty('a.ts') }))
    expect(md).toContain('涉及：a.ts')
  })

  it('干净项目给出无缺口提示与下一步指引', () => {
    const md = renderMarkdown(reportOf())
    expect(md).toContain('无收尾缺口')
    expect(md).toContain('docs/notes')
  })

  it('始终带「给下一个会话」交接段', () => {
    const md = renderMarkdown(reportOf({ dirtyFiles: dirty('a.ts') }))
    expect(md).toContain('## 给下一个会话')
    expect(md).toContain('AGENTS.md')
  })

  it('干净项目的 P 段先于交接段', () => {
    const md = renderMarkdown(reportOf())
    expect(md.indexOf('## 后续开发任务')).toBeLessThan(md.indexOf('## 给下一个会话'))
  })

  it('多任务按优先级顺序出现', () => {
    const md = renderMarkdown(reportOf({ dirtyFiles: dirty('a.ts'), ahead: 1, stashCount: 1 }))
    expect(md.indexOf('### P1')).toBeLessThan(md.indexOf('### P3'))
  })

  it('markdown 特殊字符不被改写（路径含括号）', () => {
    const md = renderMarkdown(reportOf({ otherWorktrees: ['/wt (wip) (feat/a)'] }))
    expect(md).toContain('/wt (wip) (feat/a)')
  })
})

describe('summarize', () => {
  it('干净项目一句话', () => {
    expect(summarize(reportOf())).toBe('盘点完成：无收尾缺口，报告在 .handoff/LATEST.md')
  })

  it('有任务时报总数与前三个', () => {
    const report = reportOf({
      dirtyFiles: dirty('a.ts'),
      ahead: 1,
      behind: 1,
      otherWorktrees: ['/wt (b)'],
      unmergedBranches: ['old'],
      stashCount: 1,
    })
    const text = summarize(report)
    expect(text).toContain('共 6 个任务，报告在 .handoff/LATEST.md')
    expect(text).toContain('P1 分支 feat/x')
    expect(text).toContain('P1 1 个 commit 未推送')
    expect(text).toContain('P2 落后远端')
    expect(text).not.toContain('P3')
  })

  it('恰好在截断边界（3 个任务）全列', () => {
    const report = reportOf({ dirtyFiles: dirty('a.ts'), ahead: 1, behind: 1 })
    const text = summarize(report)
    expect(text).toContain('P2 落后远端')
    expect(text).toContain('共 3 个任务')
  })
})

describe('renderMarkdown 边界', () => {
  it('无 detail 的任务不渲染涉及行', () => {
    const md = renderMarkdown(reportOf({ ahead: 1, unpushed: [] }))
    expect(md).not.toContain('涉及：')
  })

  it('actions 为空的任务只渲染标题', () => {
    const report = reportOf()
    report.tasks = [{ priority: 'P3', title: '手动补充任务', actions: [] }]
    const md = renderMarkdown(report)
    expect(md).toContain('### P3 手动补充任务')
    expect(md).not.toContain('- [ ]')
  })

  it('干净项目复选框总数为 0', () => {
    const md = renderMarkdown(reportOf())
    expect(md.split('\n').filter((l) => l.startsWith('- [ ]'))).toHaveLength(0)
  })

  it('只有 P3 任务时 summarize 列出它', () => {
    expect(summarize(reportOf({ stashCount: 1 }))).toContain('P3 stash 里有 1 条暂存记录')
  })

  it('markdown 的最后一行是插件署名', () => {
    const md = renderMarkdown(reportOf())
    expect(md.trimEnd().endsWith('处理完的任务请顺手勾选或删除。')).toBe(true)
  })

  it('标题层级：报告标题 > 小节 > 任务', () => {
    const md = renderMarkdown(reportOf({ dirtyFiles: dirty('a.ts') }))
    const h1 = md.indexOf('# 会话收尾盘点')
    const h2 = md.indexOf('## 后续开发任务')
    const h3 = md.indexOf('### P1')
    const h2b = md.indexOf('## 给下一个会话')
    expect(h1).toBeLessThan(h2)
    expect(h2).toBeLessThan(h3)
    expect(h3).toBeLessThan(h2b)
  })

  it('分支无提交时头部省略括号', () => {
    const md = renderMarkdown(reportOf({ lastCommit: '' }))
    expect(md).toContain('- 分支：feat/x')
    expect(md).not.toContain('最近提交')
  })

  it('交接段指引先读本文件与 AGENTS.md', () => {
    const md = renderMarkdown(reportOf())
    expect(md).toContain('先读本文件与仓库 AGENTS.md')
    expect(md).toContain('从最上面的任务开始处理')
  })
})
