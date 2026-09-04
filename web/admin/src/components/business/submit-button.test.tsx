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
})