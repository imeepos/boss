import { afterEach, describe, expect, it } from 'vitest'
import { Context } from '@deepseek-ai/cordis'
import type { Agent } from '@deepseek-ai/dsh-agent'
import Commands from '@deepseek-ai/dsh-commands'
import type { CommandInvocation, CommandResult } from '@deepseek-ai/dsh-commands'
import { ShellExecutor } from '@deepseek-ai/dsh-shell'
import type { ShellExecRequest, ShellExecSpec, ShellProcess, ShellRunResult } from '@deepseek-ai/dsh-shell'
import SessionHandoff from '../src/index.js'
import type { HandoffConfig } from '../src/invariant.js'

/** 真 ShellExecutor 基类 + 假后端：survey 返回罐头输出，写文件/排除记录进内存。 */
class FakeShell extends ShellExecutor {
  surveys = new Map<string, string>()
  writes: Array<{ file: string; content: string }> = []
  excludeOps: string[] = []
  surveyRuns = 0
  denied = false
  gate: Promise<void> | null = null

  constructor(ctx: Context) {
    super(ctx)
  }

  resolve(request: ShellExecRequest): ShellExecSpec {
    return {
      ...request,
      workdir: request.workdir ?? '/',
      timeoutMs: request.timeoutMs ?? 30_000,
      stdoutMaxBytes: 1_000_000,
      sandboxPolicy: undefined,
    }
  }

  async run(spec: ShellExecSpec): Promise<ShellRunResult> {
    if (this.denied) return this.result(spec, '', { mode: 'workspace-write', denied: true })
    if (spec.command.includes('###MARK')) {
      this.surveyRuns += 1
      if (this.gate) await this.gate
      return this.result(spec, this.surveys.get(spec.workdir) ?? '')
    }
    if (spec.command.includes('info/exclude')) {
      this.excludeOps.push(spec.command)
      return this.result(spec, '')
    }
    const hit = spec.command.match(/cat > '([^']+)'/)
    if (hit) {
      this.writes.push({ file: hit[1], content: spec.stdin ?? '' })
      return this.result(spec, '')
    }
    throw new Error(`unexpected command: ${spec.command}`)
  }

  start(): ShellProcess {
    throw new Error('not used')
  }

  private result(spec: ShellExecSpec, text: string, sandbox?: { mode: string; denied: boolean }): ShellRunResult {
    return {
      exitCode: 0,
      signal: null,
      timedOut: false,
      aborted: false,
      timeoutMs: spec.timeoutMs,
      stdout: { text, truncated: false },
      stderr: { text: '', truncated: false },
      ...(sandbox ? { sandbox: sandbox as ShellRunResult['sandbox'] } : {}),
    }
  }
}

const REPO_A = '/repo-a'
const REPO_B = '/repo-b'

/** 两仓库样例 survey：repo-a 脏+领先+stash，repo-b 干净。 */
function surveyA(): string {
  return [
    '###MARK toplevel',
    REPO_A,
    '###MARK branch',
    'feat/x',
    '###MARK status',
    ' M a.ts',
    '###MARK lastcommit',
    'abc1234 feat: base',
    '###MARK aheadbehind',
    '0\t2',
    '###MARK unpushed',
    'abc1234 feat: base',
    '###MARK worktrees',
    '',
    '###MARK unmerged',
    '',
    '###MARK stash',
    'stash@{0}: WIP',
  ].join('\n')
}

function surveyB(): string {
  return [
    '###MARK toplevel',
    REPO_B,
    '###MARK branch',
    'main',
    '###MARK status',
    '',
    '###MARK lastcommit',
    'def5678 chore: clean',
    '###MARK aheadbehind',
    '0\t0',
    '###MARK worktrees',
    '',
    '###MARK unmerged',
    '',
    '###MARK stash',
    '',
  ].join('\n')
}

function fakeAgent(cwd?: string): Agent {
  return { id: 'sess-1', session: { header: { cwd } } } as unknown as Agent
}

function invocation(agent: Agent): CommandInvocation {
  return {
    commandId: 'cmd-1' as CommandInvocation['commandId'],
    agent,
    rawInput: '',
    attachments: [],
    signal: new AbortController().signal,
  }
}

interface SetupOptions {
  config?: Partial<HandoffConfig>
  withCommands?: boolean
  withShell?: boolean
}

let current: Context | null = null

afterEach(async () => {
  if (current) {
    try {
      await current.dispose()
    } catch {
      // 已卸载的 fiber 二次清理可忽略
    }
  }
  current = null
})

async function setup(options: SetupOptions = {}) {
  const { config = { minIntervalMs: 0 }, withCommands = true, withShell = true } = options
  const ctx = new Context()
  current = ctx
  if (withCommands) await ctx.plugin(Commands)
  if (withShell) await ctx.plugin(FakeShell)
  await ctx.plugin(SessionHandoff, config)
  const svc = ctx.get('session-handoff') as SessionHandoff
  const fake = withShell ? (ctx.get('shell') as FakeShell) : null
  fake?.surveys.set(REPO_A, surveyA())
  fake?.surveys.set(REPO_B, surveyB())
  return { ctx, svc, fake }
}

describe('自动盘点（agent/status 驱动）', () => {
  it('running→idle 触发扫描并把报告写进 .handoff/LATEST.md', async () => {
    const { ctx, svc, fake } = await setup()
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(1)
    expect(fake?.writes).toHaveLength(1)
    expect(fake?.writes[0].file).toBe(`${REPO_A}/.handoff/LATEST.md`)
    expect(fake?.writes[0].content).toContain('# 会话收尾盘点 · repo-a')
    expect(fake?.writes[0].content).toContain('2 个 commit 未推送到 gitea')
    expect(fake?.excludeOps).toHaveLength(1)
    expect(fake?.excludeOps[0]).toContain('.handoff/')
  })

  it('idle 之外的状态不触发', async () => {
    const { ctx, svc, fake } = await setup()
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'running' })
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(0)
    expect(fake?.writes).toHaveLength(0)
  })

  it('会话没有 cwd 时跳过并留日志不炸', async () => {
    const { ctx, svc, fake } = await setup()
    ctx.emit('agent/status', { agent: fakeAgent(undefined), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(0)
  })

  it('节流窗口内同一项目只扫一次，不同项目互不影响', async () => {
    const { ctx, svc, fake } = await setup({ config: { minIntervalMs: 60_000 } })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'running' })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    ctx.emit('agent/status', { agent: fakeAgent(REPO_B), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(2)
    expect(fake?.writes.map((w) => w.file)).toEqual([`${REPO_A}/.handoff/LATEST.md`, `${REPO_B}/.handoff/LATEST.md`])
  })

  it('扫描失败不冒泡出事件总线（空 survey = 非 git 目录）', async () => {
    const { ctx, svc, fake } = await setup()
    fake?.surveys.set(REPO_A, '')
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.writes).toHaveLength(0)
  })

  it('autoExclude 关闭时不动 .git/info/exclude', async () => {
    const { ctx, svc, fake } = await setup({ config: { minIntervalMs: 0, autoExclude: false } })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.writes).toHaveLength(1)
    expect(fake?.excludeOps).toHaveLength(0)
  })
})

describe('scanProject 直调', () => {
  it('同项目并发共享同一次扫描', async () => {
    const { svc } = await setup()
    let release!: () => void
    const fake = (current!.get('shell') as FakeShell)
    fake.gate = new Promise<void>((resolve) => {
      release = resolve
    })
    const p1 = svc.scanProject(REPO_A)
    const p2 = svc.scanProject(REPO_A)
    release()
    expect(await p2).toBe(await p1)
    expect(fake.surveyRuns).toBe(1)
  })

  it('非 git 目录显式报错', async () => {
    const { svc, fake } = await setup()
    fake?.surveys.set(REPO_A, '')
    await expect(svc.scanProject(REPO_A)).rejects.toThrow(/not a git work tree/)
  })

  it('shell 服务缺失时显式报错', async () => {
    const { svc } = await setup({ withShell: false })
    await expect(svc.scanProject(REPO_A)).rejects.toThrow(/shell service unavailable/)
  })

  it('沙箱拒绝时显式报错', async () => {
    const { svc, fake } = await setup()
    fake!.denied = true
    await expect(svc.scanProject(REPO_A)).rejects.toThrow(/sandbox denied/)
  })
})

describe('/handoff 命令（真 CommandRuntime）', () => {
  it('命令注册进全局命令表', async () => {
    const { ctx } = await setup()
    const commands = ctx.get('commands')!
    const found = commands.find(fakeAgent(REPO_A), 'handoff')
    expect(found?.name).toBe('handoff')
    expect(found?.description.length).toBeGreaterThan(0)
  })

  it('handler 真执行返回成功摘要并落盘', async () => {
    const { ctx, fake } = await setup()
    const commands = ctx.get('commands')!
    const def = commands.find(fakeAgent(REPO_A), 'handoff')!
    const result: CommandResult = await def.handler(invocation(fakeAgent(REPO_A)))
    expect(result.kind).toBe('success')
    expect(result.text).toContain('共 3 个任务，报告在 .handoff/LATEST.md')
    expect(fake?.writes).toHaveLength(1)
  })

  it('干净项目返回无缺口话术', async () => {
    const { ctx } = await setup()
    const commands = ctx.get('commands')!
    const def = commands.find(fakeAgent(REPO_B), 'handoff')!
    const result: CommandResult = await def.handler(invocation(fakeAgent(REPO_B)))
    expect(result.kind).toBe('success')
    expect(result.text).toContain('无收尾缺口')
  })

  it('会话无 cwd 返回 error 话术', async () => {
    const { ctx } = await setup()
    const commands = ctx.get('commands')!
    const def = commands.find(fakeAgent(undefined), 'handoff')!
    const result: CommandResult = await def.handler(invocation(fakeAgent(undefined)))
    expect(result.kind).toBe('error')
    expect(result.text).toContain('没有关联项目目录')
  })
})

describe('装配降级', () => {
  it('commands 是必需依赖：缺失时插件等待而不半装载', async () => {
    const { ctx, fake } = await setup({ withCommands: false })
    expect(ctx.get('session-handoff')).toBeUndefined()
    expect(fake?.writes ?? []).toHaveLength(0)
  })

  it('whenSettled 在无在途扫描时立即返回', async () => {
    const { svc } = await setup()
    await expect(svc.whenSettled()).resolves.toBeUndefined()
  })

  it('同一插件实例重复扫描同一项目写两份（幂等覆盖）', async () => {
    const { svc, fake } = await setup()
    await svc.scanProject(REPO_A)
    await svc.scanProject(REPO_A)
    expect(fake?.surveyRuns).toBe(2)
    expect(fake?.writes).toHaveLength(2)
    const stripTime = (s: string) => s.split('\n').filter((l) => !l.includes('生成时间')).join('\n')
    expect(stripTime(fake!.writes[0].content)).toBe(stripTime(fake!.writes[1].content))
  })
})

describe('更多集成断言', () => {
  it('排除命令是幂等形态：先 grep 命中再追加', async () => {
    const { svc, fake } = await setup()
    await svc.scanProject(REPO_A)
    const op = fake?.excludeOps[0] ?? ''
    expect(op).toContain('grep -qxF')
    expect(op).toContain('>>')
    expect(op.indexOf('grep')).toBeLessThan(op.indexOf('>>'))
  })

  it('干净项目报告落盘含无缺口提示与交接段', async () => {
    const { ctx, svc, fake } = await setup()
    ctx.emit('agent/status', { agent: fakeAgent(REPO_B), status: 'idle' })
    await svc.whenSettled()
    const content = fake?.writes[0].content ?? ''
    expect(content).toContain('# 会话收尾盘点 · repo-b')
    expect(content).toContain('无收尾缺口')
    expect(content).toContain('## 给下一个会话')
  })

  it('minIntervalMs 为 0 时每轮 idle 都扫描（并发中的共享去重）', async () => {
    const { ctx, svc, fake } = await setup({ config: { minIntervalMs: 0 } })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(1)
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(2)
    expect(fake?.writes).toHaveLength(2)
  })

  it('同一项目两个会话同时 idle 共享一次在途扫描', async () => {
    const { ctx, svc, fake } = await setup({ config: { minIntervalMs: 60_000 } })
    let release!: () => void
    fake!.gate = new Promise<void>((resolve) => {
      release = resolve
    })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    ctx.emit('agent/status', { agent: fakeAgent(REPO_A), status: 'idle' })
    release()
    await svc.whenSettled()
    expect(fake?.surveyRuns).toBe(1)
    expect(fake?.writes).toHaveLength(1)
  })

  it('REPO_B 命令落盘到 repo-b 路径', async () => {
    const { ctx } = await setup()
    const commands = ctx.get('commands')!
    const def = commands.find(fakeAgent(REPO_B), 'handoff')!
    await def.handler(invocation(fakeAgent(REPO_B)))
    const fake = ctx.get('shell') as FakeShell
    expect(fake.writes[0].file).toBe(`${REPO_B}/.handoff/LATEST.md`)
  })

  it('扫描失败的在途 promise 结束后 inflight 清空可重试', async () => {
    const { svc, fake } = await setup()
    fake?.surveys.set(REPO_A, '')
    await expect(svc.scanProject(REPO_A)).rejects.toThrow(/not a git work tree/)
    fake?.surveys.set(REPO_A, surveyA())
    await expect(svc.scanProject(REPO_A)).resolves.toBeDefined()
    expect(fake?.writes).toHaveLength(1)
  })
})
