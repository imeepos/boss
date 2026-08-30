/**
 * session-handoff：会话收尾盘点插件。
 *
 * 监听 agent/status 的 running→idle 转换（即一轮对话收尾、没有排队工作了），
 * 自动盘点项目 git 状态，把「收尾缺口 + 后续开发任务」写入
 * <项目根>/.handoff/LATEST.md，让项目状态在会话之间不断档。
 * `/handoff` 命令可随时手动触发，不受节流限制。
 *
 * shell 与文件写入都走 ctx.shell（stdin 传文件内容，避免引号转义）；
 * 报告目录通过 .git/info/exclude 排除，不弄脏 git 状态。
 *
 * @module @ymm/dsh-session-handoff
 */
import { Service, type Context } from '@deepseek-ai/cordis'
import type { Agent, AgentStatus } from '@deepseek-ai/dsh-agent'
import type { CommandDefinition, CommandResult } from '@deepseek-ai/dsh-commands'
import type { ShellExecutor } from '@deepseek-ai/dsh-shell'
import { resolveConfig, type HandoffConfig } from './invariant.js'
import { SURVEY_SCRIPT, buildReport, parseFacts, renderMarkdown, summarize } from './survey.js'
import type { HandoffReport } from './types.js'

export const name = 'session-handoff'

export { resolveConfig, DEFAULT_CONFIG } from './invariant.js'
export type { HandoffConfig } from './invariant.js'
export { SURVEY_SCRIPT, parseFacts, classify, buildReport, renderMarkdown, summarize } from './survey.js'
export type { GitFacts, HandoffTask, TaskPriority } from './types.js'

const SURVEY_TIMEOUT_MS = 20_000

export class SessionHandoff extends Service {
  static inject = ['commands']

  private readonly config: HandoffConfig
  private readonly lastRunAt = new Map<string, number>()
  private readonly inflight = new Map<string, Promise<HandoffReport>>()

  constructor(ctx: Context, config?: Partial<HandoffConfig>) {
    super(ctx, name)
    this.config = resolveConfig(config)
    ctx.on('agent/status', ({ agent, status }) => this.onAgentStatus(agent, status))
    this.mountCommand()
  }

  /** running→idle 即「这轮对话收尾」；按项目节流后后台盘点，绝不抛回事件总线。 */
  private onAgentStatus(agent: Agent, status: AgentStatus): void {
    if (status !== 'idle') return
    const cwd = agent.session.header.cwd
    if (!cwd) {
      this.ctx.logger.warn('[session-handoff] SKIP no cwd', { sessionId: agent.id })
      return
    }
    const last = this.lastRunAt.get(cwd) ?? 0
    if (Date.now() - last < this.config.minIntervalMs) return
    this.lastRunAt.set(cwd, Date.now())
    void this.scanProject(cwd).catch((cause) => {
      this.ctx.logger.warn('[session-handoff] SCAN FAILED', { cwd, cause: String(cause) })
    })
  }

  /** 盘点一个项目；同项目并发请求共享同一次扫描。 */
  async scanProject(cwd: string): Promise<HandoffReport> {
    const running = this.inflight.get(cwd)
    if (running) return running
    const task = this.doScan(cwd).finally(() => this.inflight.delete(cwd))
    this.inflight.set(cwd, task)
    return task
  }

  /** 等待全部在途扫描结束（失败也算结束）；测试与优雅关闭用。 */
  async whenSettled(): Promise<void> {
    await Promise.allSettled([...this.inflight.values()])
  }

  private async doScan(cwd: string): Promise<HandoffReport> {
    const facts = parseFacts(await this.runShell(SURVEY_SCRIPT, cwd))
    if (!facts.toplevel) throw new Error(`not a git work tree: ${cwd}`)
    const report = buildReport(facts, this.config)
    await this.persist(facts.toplevel, report)
    this.ctx.logger.info('[session-handoff] report written', { project: report.project, tasks: report.tasks.length })
    return report
  }

  private async runShell(command: string, workdir: string, stdin?: string): Promise<string> {
    const shell: ShellExecutor | undefined = this.ctx.get('shell')
    if (!shell) {
      this.ctx.logger.warn('[session-handoff] shell service missing; SKIP')
      throw new Error('shell service unavailable')
    }
    const spec = shell.resolve({ command, workdir, timeoutMs: SURVEY_TIMEOUT_MS, stdin })
    const done = await shell.run(spec)
    if (done.sandbox?.denied) throw new Error(`sandbox denied: ${command.slice(0, 80)}`)
    return done.stdout.text
  }

  private async persist(toplevel: string, report: HandoffReport): Promise<void> {
    const dir = `${toplevel}/${this.config.outputDir}`
    const file = `${dir}/${this.config.fileName}`
    await this.runShell(`mkdir -p ${shquote(dir)} && cat > ${shquote(file)}`, toplevel, renderMarkdown(report))
    if (this.config.autoExclude) await this.ensureExcluded(toplevel)
  }

  private async ensureExcluded(toplevel: string): Promise<void> {
    const exclude = `${toplevel}/.git/info/exclude`
    const entry = `${this.config.outputDir}/`
    const cmd = `grep -qxF ${shquote(entry)} ${shquote(exclude)} 2>/dev/null || printf '%s\\n' ${shquote(entry)} >> ${shquote(exclude)}`
    await this.runShell(cmd, toplevel)
  }

  private mountCommand(): void {
    const commands = this.ctx.get('commands')
    if (!commands) {
      this.ctx.logger.warn('[session-handoff] commands missing; /handoff unavailable')
      return
    }
    this.ctx.effect(() => commands.register({
      name: 'handoff',
      description: '盘点项目收尾状态并生成后续开发任务',
      handler: async (invocation) => this.handleCommand(invocation.agent),
    }), 'session-handoff-command')
  }

  private async handleCommand(agent: Agent): Promise<CommandResult> {
    const cwd = agent.session.header.cwd
    if (!cwd) return { kind: 'error', text: '当前会话没有关联项目目录，无法盘点' }
    const report = await this.scanProject(cwd)
    return { kind: 'success', text: summarize(report) }
  }
}

function shquote(path: string): string {
  return `'${path.replaceAll("'", `'\\''`)}'`
}

export default SessionHandoff
