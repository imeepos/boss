/**
 * session-handoff 的公开类型：git 盘点事实、任务、报告。
 * @module @ymm/dsh-session-handoff/types
 */

export type TaskPriority = 'P0' | 'P1' | 'P2' | 'P3'

/** `git status --porcelain` 的一行。 */
export interface DirtyFile {
  /** 两位状态码，如 ` M`、`??`、`R `。 */
  code: string
  /** 文件路径（重命名取新路径）。 */
  path: string
}

/** 一次盘点采集到的 git 事实（全部来自单次 survey 脚本输出）。 */
export interface GitFacts {
  /** 项目根（`git rev-parse --show-toplevel`）；空串表示不是 git 仓库。 */
  toplevel: string
  /** 当前分支名；detached HEAD 时为空串。 */
  branch: string
  /** 未提交改动清单。 */
  dirtyFiles: DirtyFile[]
  /** 最近一次提交，格式 `<hash> <subject>`。 */
  lastCommit: string
  /** 是否配置了上游分支。 */
  hasUpstream: boolean
  /** 领先上游的 commit 数。 */
  ahead: number
  /** 落后上游的 commit 数。 */
  behind: number
  /** 未推送 commit 的 oneline 列表（最多 10 条）。 */
  unpushed: string[]
  /** 主树之外的 worktree，格式 `<path> (<branch>)`。 */
  otherWorktrees: string[]
  /** 未合入当前 HEAD 的本地分支名。 */
  unmergedBranches: string[]
  /** stash 记录条数。 */
  stashCount: number
}

/** 一条后续开发任务。 */
export interface HandoffTask {
  priority: TaskPriority
  title: string
  detail?: string
  /** 可直接执行的动作建议。 */
  actions: string[]
}

/** 一份收尾盘点报告。 */
export interface HandoffReport {
  generatedAt: string
  /** 项目目录名。 */
  project: string
  branch: string
  lastCommit: string
  facts: GitFacts
  tasks: HandoffTask[]
  /** 报告落盘位置，如 `.handoff/LATEST.md`。 */
  outputHint: string
}
