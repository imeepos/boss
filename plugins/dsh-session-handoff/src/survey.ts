/**
 * 盘点纯函数层：survey 脚本输出 → git 事实 → 任务清单 → markdown 报告。
 * 全部无副作用，单测直接覆盖。
 * @module @ymm/dsh-session-handoff/survey
 */

import type { HandoffConfig } from './invariant.js'
import type { DirtyFile, GitFacts, HandoffReport, HandoffTask } from './types.js'

/** 一次盘点要跑的全部 git 命令，用 `###MARK` 分段；任何一段失败都不阻断后续。 */
export const SURVEY_SCRIPT = [
  'set +e',
  'echo "###MARK toplevel"; git rev-parse --show-toplevel',
  'echo "###MARK branch"; git branch --show-current',
  'echo "###MARK status"; git status --porcelain',
  'echo "###MARK lastcommit"; git log -1 --format="%h %s"',
  'echo "###MARK aheadbehind"; git rev-list --left-right --count "@{upstream}...HEAD"',
  'echo "###MARK unpushed"; git log --oneline "@{upstream}..HEAD" -n 10',
  'echo "###MARK worktrees"; git worktree list --porcelain',
  'echo "###MARK unmerged"; git branch --no-merged',
  'echo "###MARK stash"; git stash list',
  'echo "###MARK end"',
].join('\n')

/** 按 `###MARK <key>` 行把脚本输出切成段。 */
export function splitSurvey(raw: string): Record<string, string> {
  const sections: Record<string, string> = {}
  let key = ''
  let buf: string[] = []
  for (const line of raw.split('\n')) {
    const hit = line.match(/^###MARK (\w+)\s*$/)
    if (!hit) {
      if (key) buf.push(line)
      continue
    }
    if (key) sections[key] = buf.join('\n')
    key = hit[1]
    buf = []
  }
  if (key) sections[key] = buf.join('\n')
  return sections
}

function lines(text: string | undefined): string[] {
  return (text ?? '').split('\n').map((l) => l.trim()).filter((l) => l.length > 0)
}

/** porcelain 段专用：保留行首空格（状态码对位），只去 \r，过滤空行。 */
function rawLines(text: string | undefined): string[] {
  return (text ?? '').split('\n').map((l) => l.replace(/\r$/, '')).filter((l) => l.trim().length > 0)
}

/** 解析 porcelain 一行：`XY path` 或 `XY old -> new`（取新路径）。 */
export function parseStatusLine(line: string): DirtyFile | null {
  if (line.length < 4) return null
  const rest = line.slice(3)
  const raw = rest.includes(' -> ') ? rest.split(' -> ').pop()! : rest
  const path = raw.startsWith('"') && raw.endsWith('"') ? raw.slice(1, -1) : raw
  return { code: line.slice(0, 2), path }
}

/** `--left-right --count` 输出为 `<behind>\\t<ahead>`；空输出表示没有上游。 */
export function parseAheadBehind(text: string): { ahead: number; behind: number; hasUpstream: boolean } {
  const trimmed = text.trim()
  if (!trimmed) return { ahead: 0, behind: 0, hasUpstream: false }
  const parts = trimmed.split(/\s+/).map(Number)
  if (parts.length !== 2 || parts.some((n) => !Number.isFinite(n))) {
    return { ahead: 0, behind: 0, hasUpstream: true }
  }
  return { behind: parts[0], ahead: parts[1], hasUpstream: true }
}

/** porcelain worktree 清单；主树按路径剔除，返回 `<path> (<branch>)`。 */
export function parseWorktrees(text: string, mainTree: string): string[] {
  const found: string[] = []
  let current: string | null = null
  for (const line of text.split('\n')) {
    if (line.startsWith('worktree ')) {
      current = line.slice('worktree '.length).trim()
    } else if (line.startsWith('branch ') && current && current !== mainTree) {
      const branch = line.slice('branch '.length).trim().replace('refs/heads/', '')
      found.push(`${current} (${branch})`)
    }
  }
  return found
}

function stripBranchMark(line: string): string {
  return line.replace(/^[*+]\s*/, '').trim()
}

/** survey 输出 → 结构化事实。 */
export function parseFacts(raw: string): GitFacts {
  const s = splitSurvey(raw)
  const aheadBehind = parseAheadBehind(s.aheadbehind ?? '')
  const toplevel = (s.toplevel ?? '').trim()
  return {
    toplevel,
    branch: (s.branch ?? '').trim(),
    dirtyFiles: rawLines(s.status).map(parseStatusLine).filter((f) => f !== null),
    lastCommit: (s.lastcommit ?? '').trim(),
    hasUpstream: aheadBehind.hasUpstream,
    ahead: aheadBehind.ahead,
    behind: aheadBehind.behind,
    unpushed: lines(s.unpushed),
    otherWorktrees: parseWorktrees(s.worktrees ?? '', toplevel),
    unmergedBranches: lines(s.unmerged).map(stripBranchMark).filter((b) => b.length > 0),
    stashCount: lines(s.stash).length,
  }
}

/** 事实 → 任务清单，按 P0→P3 排序；判定基准对齐仓库收尾协议。 */
export function classify(facts: GitFacts, config: HandoffConfig): HandoffTask[] {
  const tasks: HandoffTask[] = []
  if (facts.dirtyFiles.length > 0) collectDirty(tasks, facts, config)
  collectRemote(tasks, facts, config)
  if (facts.otherWorktrees.length > 0) collectWorktrees(tasks, facts)
  collectUnmerged(tasks, facts, config)
  if (facts.stashCount > 0) collectStash(tasks, facts)
  return tasks.sort((a, b) => a.priority.localeCompare(b.priority))
}

function collectDirty(tasks: HandoffTask[], facts: GitFacts, config: HandoffConfig): void {
  const count = facts.dirtyFiles.length
  const preview = previewOf(facts.dirtyFiles.map((f) => f.path))
  if (facts.branch === config.mainBranch) {
    tasks.push({
      priority: 'P0',
      title: `主分支 ${config.mainBranch} 上有 ${count} 个未提交改动（仓库红线：禁止主分支直接改代码）`,
      detail: `涉及：${preview}`,
      actions: ['建 worktree 分支挪走改动后再提交', '确认是临时产物则直接清理'],
    })
    return
  }
  tasks.push({
    priority: 'P1',
    title: `分支 ${facts.branch} 有 ${count} 个未提交文件，存在丢失风险`,
    detail: `涉及：${preview}`,
    actions: ['git add <files> && git commit（type(scope): subject）', '或 git stash 暂存'],
  })
}

function collectRemote(tasks: HandoffTask[], facts: GitFacts, config: HandoffConfig): void {
  if (!facts.hasUpstream && facts.branch !== config.mainBranch) {
    tasks.push({
      priority: 'P1',
      title: `分支 ${facts.branch} 没有上游，commit 只存在于本地`,
      actions: [`git push -u ${config.remote} ${facts.branch}`],
    })
    return
  }
  if (facts.ahead > 0) {
    tasks.push({
      priority: 'P1',
      title: `${facts.ahead} 个 commit 未推送到 ${config.remote}`,
      detail: facts.unpushed.length > 0 ? previewOf(facts.unpushed, 3) : undefined,
      actions: [`git push ${config.remote} ${facts.branch}`],
    })
  }
  if (facts.behind > 0) {
    tasks.push({
      priority: 'P2',
      title: `落后远端 ${facts.behind} 个提交，合并前需先反向同步`,
      actions: [`git merge ${config.remote}/${config.mainBranch}`],
    })
  }
}

function collectWorktrees(tasks: HandoffTask[], facts: GitFacts): void {
  tasks.push({
    priority: 'P2',
    title: `${facts.otherWorktrees.length} 个 worktree 残留，先核对归属再清理`,
    detail: previewOf(facts.otherWorktrees),
    actions: ['完工的按收尾四步清理：push → ff-only 合并 → worktree remove → 删分支'],
  })
}

function collectUnmerged(tasks: HandoffTask[], facts: GitFacts, config: HandoffConfig): void {
  const others = facts.unmergedBranches.filter((b) => b !== facts.branch && b !== config.mainBranch)
  if (others.length === 0) return
  tasks.push({
    priority: 'P2',
    title: `${others.length} 个分支未合入 ${config.mainBranch}`,
    detail: previewOf(others),
    actions: ['逐个确认：已完工的当天合并回主分支，废弃的删除'],
  })
}

function collectStash(tasks: HandoffTask[], facts: GitFacts): void {
  tasks.push({
    priority: 'P3',
    title: `stash 里有 ${facts.stashCount} 条暂存记录`,
    actions: ['确认是否还需要：git stash list / git stash pop / git stash drop'],
  })
}

function previewOf(items: string[], cap = 5): string {
  const head = items.slice(0, cap).join('、')
  return items.length > cap ? `${head} 等 ${items.length} 项` : head
}

/** 事实 → 报告。now 可注入便于测试。 */
export function buildReport(facts: GitFacts, config: HandoffConfig, now = new Date()): HandoffReport {
  const project = facts.toplevel.split('/').filter(Boolean).pop() ?? facts.toplevel
  return {
    generatedAt: now.toISOString(),
    project,
    branch: facts.branch,
    lastCommit: facts.lastCommit,
    facts,
    tasks: classify(facts, config),
    outputHint: `${config.outputDir}/${config.fileName}`,
  }
}

/** 报告 → markdown。 */
export function renderMarkdown(report: HandoffReport): string {
  const head = [
    `# 会话收尾盘点 · ${report.project}`,
    '',
    `- 生成时间：${report.generatedAt}`,
    `- 分支：${report.branch}${report.lastCommit ? `（最近提交：${report.lastCommit}）` : ''}`,
    `- 与远端：ahead ${report.facts.ahead} / behind ${report.facts.behind} / 未提交 ${report.facts.dirtyFiles.length} 个文件`,
    '',
  ]
  return [...head, ...taskSection(report), '', briefSection()].join('\n')
}

function taskSection(report: HandoffReport): string[] {
  const out = ['## 后续开发任务（按优先级）', '']
  if (report.tasks.length === 0) {
    out.push('无收尾缺口——工作区干净、远端同步。下一任务可从 docs/notes、backlog 或 devloop 账本挑起。', '')
    return out
  }
  for (const task of report.tasks) {
    out.push(`### ${task.priority} ${task.title}`, '')
    if (task.detail) out.push(task.detail, '')
    for (const action of task.actions) out.push(`- [ ] ${action}`)
    out.push('')
  }
  return out
}

function briefSection(): string[] {
  return [
    '## 给下一个会话',
    '',
    '开工前先读本文件与仓库 AGENTS.md，从最上面的任务开始处理。',
    '本文件由 session-handoff 插件自动生成；处理完的任务请顺手勾选或删除。',
  ]
}

/** 命令回显用的一句话摘要。 */
export function summarize(report: HandoffReport): string {
  if (report.tasks.length === 0) return `盘点完成：无收尾缺口，报告在 ${report.outputHint}`
  const top = report.tasks.slice(0, 3).map((t) => `${t.priority} ${t.title}`)
  return [`盘点完成：共 ${report.tasks.length} 个任务，报告在 ${report.outputHint}`, ...top].join('\n')
}
