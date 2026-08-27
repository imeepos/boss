// 回访评价页 i18n 骨架回归:SSR 渲染标题/列名/筛选占位/分页文案,
// 防止 i18n key 漂移与硬编码回退。仅走 renderToStaticMarkup
// (项目 vitest environment=node,无 @testing-library)。
import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ConfirmProvider } from '../../../components/ConfirmDialog'
import FeedbackPage from './index'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn(() => Promise.resolve({ items: [] })) }))

vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { loading: '加载中…', confirmDialog: { title: '操作确认', ok: '确认', cancel: '取消' } },
    pages: {
      feedbackPage: {
        title: '回访评价',
        desc: 'desc',
        filterWorker: '师傅',
        filterWorkerPh: '按姓名/工号筛选',
        filterReviewOnly: '只看待复核',
        columns: ['评价 ID', '客户', '师傅', '工单', '装维队', '服务区域', '企业', '评分', '复核'],
        scoreFmt: '{score}/5',
        needReviewYes: '待复核',
        needReviewNo: '已处理',
        review: '复核',
        reviewConfirm: '确认复核?',
        actionFail: '操作失败',
        total: '共 {count} 条',
        empty: '暂无评价',
        loadFail: '加载失败',
        prev: '上一页', next: '下一页', perPage: '条/页',
        rangeText: '第 {from}-{to} 条', jumpText: '跳至', pageUnit: '页',
      },
      audit: { refresh: '刷新' },
    },
  }),
}))

describe('FeedbackPage SSR 文案/列名骨架', () => {
  it('列名与标题来自 i18n(防硬编码)', () => {
    const html = renderToStaticMarkup(
      <ConfirmProvider><FeedbackPage /></ConfirmProvider>,
    )
    expect(html).toContain('回访评价')
    expect(html).toContain('评价 ID')
    expect(html).toContain('客户')
    expect(html).toContain('师傅')
    expect(html).toContain('工单')
    expect(html).toContain('装维队')
    expect(html).toContain('服务区域')
    expect(html).toContain('企业')
    expect(html).toContain('评分')
    expect(html).toContain('复核')
    expect(html).toContain('按姓名/工号筛选')
    expect(html).toContain('只看待复核')
    expect(html).toContain('刷新')
    expect(html).toContain('暂无评价')
  })
})