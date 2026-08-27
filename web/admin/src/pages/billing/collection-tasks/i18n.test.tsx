// 催收任务队列页 i18n 骨架回归:SSR 渲染标题/状态过滤/列名/动作按钮文案,
// 防止 i18n key 漂移与硬编码回退(原页曾直接渲染 'PENDING'/'DOING'/'DONE'/'FAILED')。
// 仅走 renderToStaticMarkup(项目 vitest environment=node,无 @testing-library)。
import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ConfirmProvider } from '../../../components/ConfirmDialog'
import CollectionTasksPage from './index'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn(() => Promise.resolve({ items: [] })) }))

vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { loading: '加载中…', confirmDialog: { title: '操作确认', ok: '确认', cancel: '取消' } },
    pages: {
      collectionTasksPage: {
        title: '催收任务队列',
        desc: 'AR 域 ar_collection_tasks 人工催收队列描述',
        statuses: { PENDING: '待处理', DOING: '处理中', DONE: '已完成', FAILED: '失败' },
        columns: ['ID', '客户', '欠费/账龄', '优先级', '状态', '到期', '操作'],
        actionStart: '开始',
        actionDone: '完成',
        actionFail: '失败',
        actionConfirm: '确认将该任务置为 {status}?',
        actionFailMsg: '操作失败',
        empty: '暂无催收任务',
        loadFail: '加载失败',
        total: '共 {count} 条',
        prev: '上一页', next: '下一页', perPage: '条/页',
        rangeText: '第 {from}-{to} 条', jumpText: '跳至', pageUnit: '页',
      },
      audit: { refresh: '刷新' },
    },
  }),
}))

describe('CollectionTasksPage SSR 文案骨架', () => {
  it('标题/列名/状态过滤全部走 i18n(防硬编码 PENDING/DOING/DONE/FAILED)', () => {
    const html = renderToStaticMarkup(
      <ConfirmProvider><CollectionTasksPage /></ConfirmProvider>,
    )
    expect(html).toContain('催收任务队列')
    // 状态过滤按钮文案必须来自 i18n.statuses,不允许出现英文原始状态名作为按钮文本
    expect(html).toContain('待处理')
    expect(html).toContain('处理中')
    expect(html).toContain('已完成')
    expect(html).toContain('失败')
    // 列名走 i18n
    expect(html).toContain('欠费/账龄')
    expect(html).toContain('优先级')
    expect(html).toContain('到期')
    // 刷新 + 空态文案
    expect(html).toContain('刷新')
    expect(html).toContain('暂无催收任务')
  })
})