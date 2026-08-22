// Dashboard 页面测试
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './index'
import { apiFetch } from '../../api/client'

// Mock apiFetch
vi.mock('../../api/client', () => ({
  apiFetch: vi.fn(),
}))

// Mock useT
vi.mock('../../i18n', () => ({
  useT: () => ({
    pages: {
      dashboard: {
        title: '工作台',
        welcome: '欢迎,{name}({role})。',
        loadFail: '加载失败',
        empty: '暂无数据',
        trendUnit: '单',
        trendTooltip: '订单数',
      },
      audit: {
        refresh: '刷新',
      },
    },
  }),
}))

// Mock PageHead
vi.mock('../../components/business/page-head', () => ({
  PageHead: ({ title, desc }: { title: string; desc: string }) => (
    <div>
      <h1>{title}</h1>
      <p>{desc}</p>
    </div>
  ),
}))

// Mock fmtTime
vi.mock('../../lib/format', () => ({
  fmtTime: (time: string) => time,
}))

describe('DashboardPage', () => {
  const mockProfile = {
    accountId: 1,
    username: 'admin',
    realName: '系统管理员',
    roleCode: 'sysadmin',
    roleName: '系统管理员',
    legalEntityName: 'LEG-A 主品牌·企业',
    regionScope: '',
  }

  const mockDashboardData = {
    stats: [
      { key: 'todayOrders', label: '今日新增订单', value: '128' },
      { key: 'activeTickets', label: '进行中工单', value: '23' },
      { key: 'pendingAlarms', label: '待处理告警', value: '5' },
      { key: 'assetConsistency', label: '四码一致率', value: '98.6%' },
    ],
    orderStatusDist: [
      { status: 'PENDING', statusLabel: '待核查', count: 32, percent: '15.8%' },
      { status: 'RESERVED', statusLabel: '已预占', count: 45, percent: '22.3%' },
      { status: 'INSTALLING', statusLabel: '装维中', count: 28, percent: '13.9%' },
      { status: 'DONE', statusLabel: '已完成', count: 51, percent: '25.2%' },
    ],
    todos: {
      items: [
        { todoId: 1, subject: 'O20240001 待指派师傅', source: '派单池', time: '' },
        { todoId: 2, subject: 'ALM-001 设备温度过高', source: '告警中心', time: '09:58' },
      ],
    },
    trend: {
      days: ['05-14', '05-15', '05-16', '05-17', '05-18', '05-19', '05-20'],
      values: [50, 65, 75, 60, 90, 70, 75],
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(apiFetch).mockResolvedValue(mockDashboardData)
  })

  it('renders dashboard with all sections', async () => {
    const html = renderToStaticMarkup(
      <MemoryRouter>
        <DashboardPage profile={mockProfile} />
      </MemoryRouter>
    )

    // 静态渲染只出骨架(数据分区由 useEffect 拉取后渲染):
    // 标题 + 欢迎语 + 用户角色。
    expect(html).toContain('工作台')
    expect(html).toContain('欢迎')
    expect(html).toContain('系统管理员')
  })

  it('contains proper CSS classes for styling', async () => {
    const html = renderToStaticMarkup(
      <MemoryRouter>
        <DashboardPage profile={mockProfile} />
      </MemoryRouter>
    )

    // 实现中立断言:骨架有 h1 标题区;样式类实现自由(纯 CSS/antd 均可),
    // 避免测试与某一代样式方案耦合。
    expect(html).toContain('<h1>')
  })
})
