import { afterEach, describe, expect, it } from 'vitest'
import { Context } from '@deepseek-ai/cordis'
import type { Agent } from '@deepseek-ai/dsh-agent'
import Commands from '@deepseek-ai/dsh-commands'
import { FakeShellPlugin, surveyFor, REPO } from './fixtures.js'
import SessionHandoff from '../src/index.js'
import type { HandoffConfig } from '../src/invariant.js'

function fakeAgent(cwd?: string): Agent {
  return { id: 'sess-path', session: { header: { cwd } } } as unknown as Agent
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

async function setup(config: Partial<HandoffConfig>) {
  const ctx = new Context()
  current = ctx
  await ctx.plugin(Commands)
  await ctx.plugin(FakeShellPlugin)
  const fake = ctx.get('shell') as InstanceType<typeof FakeShellPlugin>
  fake.surveys.set(REPO, surveyFor(REPO, 'feat/paths'))
  await ctx.plugin(SessionHandoff, { minIntervalMs: 0, ...config })
  return { ctx, fake, svc: ctx.get('session-handoff') as SessionHandoff }
}

describe('配置驱动落盘路径', () => {
  it('自定义 outputDir 改变报告目录', async () => {
    const { svc, fake } = await setup({ outputDir: 'handoff' })
    await svc.scanProject(REPO)
    expect(fake.writes[0].file).toBe(`${REPO}/handoff/LATEST.md`)
    expect(fake.excludeOps[0]).toContain('handoff/')
  })

  it('自定义 fileName 改变报告文件名', async () => {
    const { svc, fake } = await setup({ fileName: 'NOW.md' })
    await svc.scanProject(REPO)
    expect(fake.writes[0].file).toBe(`${REPO}/.handoff/NOW.md`)
  })

  it('outputDir 与 fileName 同时自定义', async () => {
    const { svc, fake } = await setup({ outputDir: 'docs/handoff', fileName: 'SURVEY.md' })
    await svc.scanProject(REPO)
    expect(fake.writes[0].file).toBe(`${REPO}/docs/handoff/SURVEY.md`)
  })

  it('自定义 mainBranch 改变 P0 判定', async () => {
    const { svc, fake } = await setup({ mainBranch: 'feat/paths' })
    const report = await svc.scanProject(REPO)
    expect(report.tasks.map((t) => t.priority)).toEqual(['P0', 'P1'])
    expect(fake.writes[0].content).toContain('主分支 feat/paths')
  })

  it('自定义 remote 改变推送话术', async () => {
    const { svc } = await setup({ remote: 'origin' })
    const report = await svc.scanProject(REPO)
    expect(report.tasks.some((t) => t.title.includes('未推送到 origin'))).toBe(true)
  })

  it('报告内容内嵌配置产物 outputHint', async () => {
    const { svc, fake } = await setup({ outputDir: 'h', fileName: 'X.md' })
    const report = await svc.scanProject(REPO)
    expect(report.outputHint).toBe('h/X.md')
    expect(fake.writes[0].content).toContain('- 分支：feat/paths')
  })
})

describe('命令面补充断言', () => {
  it('命令描述非空且不含斜杠前缀', async () => {
    const { ctx } = await setup({})
    const def = ctx.get('commands')!.find(fakeAgent(REPO), 'handoff')!
    expect(def.description.length).toBeGreaterThan(4)
    expect(def.description.startsWith('/')).toBe(false)
  })

  it('命令表里只有 handoff 一条本插件命令', async () => {
    const { ctx } = await setup({})
    const names = ctx.get('commands')!.list(fakeAgent(REPO)).map((d) => d.name)
    expect(names).toContain('handoff')
    expect(names.filter((n) => n === 'handoff')).toHaveLength(1)
  })

  it('handler 两次调用各自落盘（节流不限制命令）', async () => {
    const { ctx, fake } = await setup({ minIntervalMs: 60_000 })
    const def = ctx.get('commands')!.find(fakeAgent(REPO), 'handoff')!
    await def.handler({
      commandId: 'c1' as never,
      agent: fakeAgent(REPO),
      rawInput: '',
      attachments: [],
      signal: new AbortController().signal,
    })
    await def.handler({
      commandId: 'c2' as never,
      agent: fakeAgent(REPO),
      rawInput: '',
      attachments: [],
      signal: new AbortController().signal,
    })
    expect(fake.surveyRuns).toBe(2)
  })
})
