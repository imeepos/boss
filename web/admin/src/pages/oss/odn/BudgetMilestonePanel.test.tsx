// 预算里程碑面板抽屉化回归(A1 口径):面板正文不再渲染任何表单控件,
// 追加/编辑入口均为按钮;只读态保留锁定文案。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { BudgetMilestonePanel } from './BudgetMilestonePanel'

const project = { id: 7, status: 'PENDING', budgetAmount: 120000, settledAmount: 30000 }

describe('BudgetMilestonePanel 面板正文无内联表单', () => {
  it('PENDING 可改态:静态渲染零 input 控件,入口为追加/编辑预算按钮', () => {
    const html = renderToStaticMarkup(<BudgetMilestonePanel project={project} onChanged={() => {}} />)
    expect(html).not.toContain('<input')
    expect(html).toContain('追加里程碑')
    expect(html).toContain('编辑预算')
    expect(html).toContain('预算金额:')
  })

  it('竣工锁定态:显示锁定文案,无编辑入口', () => {
    const html = renderToStaticMarkup(
      <BudgetMilestonePanel project={{ ...project, status: 'ACCEPTED' }} onChanged={() => {}} />)
    expect(html).not.toContain('<input')
    expect(html).not.toContain('编辑预算')
    expect(html).toContain('(已竣工锁定)')
  })
})
