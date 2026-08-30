import { Context } from '@deepseek-ai/cordis'
import { ShellExecutor } from '@deepseek-ai/dsh-shell'
import type { ShellExecRequest, ShellExecSpec, ShellProcess, ShellRunResult } from '@deepseek-ai/dsh-shell'

export const REPO = '/repo-paths'

/**
 * 集成测试共享的假 shell：继承真 ShellExecutor 基类（fake 只换执行后端）。
 * survey 命令返回罐头输出；写文件/排除命令记录进内存。
 */
export class FakeShellPlugin extends ShellExecutor {
  surveys = new Map<string, string>()
  writes: Array<{ file: string; content: string }> = []
  excludeOps: string[] = []
  surveyRuns = 0
  denied = false
  gate: Promise<void> | null = null

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

/** 一个"领先 2、有脏改"的标准样例 survey。 */
export function surveyFor(toplevel: string, branch: string): string {
  return [
    '###MARK toplevel',
    toplevel,
    '###MARK branch',
    branch,
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
    '',
  ].join('\n')
}
