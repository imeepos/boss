// 里程碑抽屉回归:双模式标题/字段渲染与表单初值回填(initialMilestoneForm 纯函数口径)。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { initialMilestoneForm, MilestoneDrawer, type Milestone } from './MilestoneDrawer'

const milestone: Milestone = {
  id: 3, projectId: 7, name: '主干光缆敷设完成', plannedDate: '2026-10-01',
  status: 'PENDING', createdAt: '2026-09-01T00:00:00Z',
}
const props = (editing: Milestone | null) => ({
  target: { projectId: 7, editing }, onClose: () => {}, onSaved: () => {},
})

describe('initialMilestoneForm 初值', () => {
  it('create 空表单', () => {
    expect(initialMilestoneForm(null)).toEqual({ name: '', plannedDate: '' })
  })
  it('edit 回填行值,plannedDate 缺省补空', () => {
    expect(initialMilestoneForm(milestone)).toEqual({ name: '主干光缆敷设完成', plannedDate: '2026-10-01' })
    expect(initialMilestoneForm({ ...milestone, plannedDate: undefined }).plannedDate).toBe('')
  })
})

describe('MilestoneDrawer 双模式渲染', () => {
  it('create 模式:追加标题 + 名称/计划完成日字段', () => {
    const html = renderToStaticMarkup(<MilestoneDrawer {...props(null)} />)
    expect(html).toContain('追加里程碑')
    expect(html).not.toContain('编辑里程碑')
    expect(html).toContain('名称')
    expect(html).toContain('计划完成日')
  })

  it('edit 模式:编辑标题 + 行值回填输入框', () => {
    const html = renderToStaticMarkup(<MilestoneDrawer {...props(milestone)} />)
    expect(html).toContain('编辑里程碑')
    expect(html).toContain('value="主干光缆敷设完成"')
  })
})
