// SubmitButton 回归:四态渲染对应文案与 data-submit-state 标记,loading 态禁点。
import { describe, it, expect } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { SubmitButton } from './submit-button'
import { LocaleProvider } from '../../i18n/context'

const labels = { idle: '新建', loading: '提交中…', success: '创建成功', failed: '创建失败' }

describe('SubmitButton', () => {
  it.each(['idle', 'loading', 'success', 'failed'] as const)('%s 态渲染对应文案与状态标记', (s) => {
    const html = renderToStaticMarkup(
      <LocaleProvider><SubmitButton state={s} labels={labels} onClick={() => {}} /></LocaleProvider>,
    )
    expect(html).toContain(labels[s])
    expect(html).toContain('data-submit-state="' + s + '"')
  })

  it('loading 态按钮 disabled 且含 spinner', () => {
    const html = renderToStaticMarkup(
      <LocaleProvider><SubmitButton state="loading" labels={labels} /></LocaleProvider>,
    )
    expect(html).toContain('disabled')
    expect(html).toContain('animate-spin')
  })

  it('danger 显式声明:idle/loading 危险色实底,success/failed 语义色不变', () => {
    const dangerIdle = renderToStaticMarkup(
      <LocaleProvider><SubmitButton state="idle" labels={labels} danger /></LocaleProvider>,
    )
    expect(dangerIdle).toContain('bg-[var(--color-danger)]')
    const dangerFailed = renderToStaticMarkup(
      <LocaleProvider><SubmitButton state="failed" labels={labels} danger /></LocaleProvider>,
    )
    // failed 本就是危险色,断言语义色不被 danger 分支改写(仍走 STATE_CLS.failed)
    expect(dangerFailed).toContain('data-submit-state="failed"')
    expect(dangerFailed).toContain('bg-[var(--color-danger)]')
    // 未声明 danger 时 idle 维持品牌实底
    const plainIdle = renderToStaticMarkup(
      <LocaleProvider><SubmitButton state="idle" labels={labels} /></LocaleProvider>,
    )
    expect(plainIdle).toContain('bg-[var(--shell-fab-bg)]')
  })
})