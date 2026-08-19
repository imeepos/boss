// Profile 页面测试:各分区路由骨架渲染(SSR,apiFetch 不触发)。
import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import ProfilePage from './index'
import { ProfileContext } from '../../layouts/profile'

vi.mock('../../api/client', () => ({ apiFetch: vi.fn() }))
vi.mock('../../i18n', () => ({
  useT: () => ({
    pages: {
      profile: {
        cancel: '取消', save: '保存',
        password: { title: '安全设置', desc: '修改登录密码前需要验证旧密码', old: '旧密码', next: '新密码', confirm: '确认新密码', submit: '更新密码', passwordPending: '待接入', success: '密码已更新', fail: '密码更新失败', mismatch: '两次输入的新密码不一致' },
        securityProtection: { title: '其他安全设置', loginProtection: '登录保护', loginHistory: '最近登录记录', pending: '接口待接入' },
        apiKey: { title: 'API Key 管理', desc: '用于自动化调用', placeholder: '占位设计', loadFail: '加载失败', denied: '当前角色无权管理 API Key', empty: '暂无密钥', name: '名称', key: 'Key 前缀', lastUsed: '最近使用', status: '状态', namePlaceholder: '输入密钥用途', create: '创建 Key', neverUsed: '从未使用', active: '启用', revoke: '撤销', securityTip: '安全提示' },
        audit: { title: '我的操作审计', desc: '仅展示当前账号产生的审计记录', empty: '暂无审计记录', emptyDesc: '最近 20 条操作记录', loadFail: '加载失败' },
        myData: { title: '我的工作', desc: '仅展示当前后台账号负责的业务记录', orders: '我负责的订单', ordersDesc: '订单', bills: '我处理的账务任务', billsDesc: '账务', service: '我处理的工单', serviceDesc: '工单', messages: '工作通知', messagesDesc: '通知', audit: '我的操作记录', auditDesc: '审计', permissions: '我的权限', permissionsDesc: '权限' },
        personal: { title: '基本资料', desc: '维护当前登录账号的可编辑资料', badge: '当前账号', username: '登录账号', realName: '姓名', phone: '手机号', phonePlaceholder: '请输入手机号', saved: '已保存', saveFail: '保存失败', role: '角色', company: '所属公司', dataScope: '数据范围', unassigned: '未分配', allScope: '全集团' },
        permissions: { title: '我的权限', desc: '当前账号的角色与数据范围,仅供查看', role: '当前角色', company: '所属公司', scope: '数据范围' },
      },
      apikey: { revoked: '已撤销', plainOnce: '完整密钥仅展示一次' },
      company: { cancel: '取消' },
    },
  }),
}))

const profile = {
  accountId: 1, username: 'boss', realName: '老板', phone: '13800000000',
  roleCode: 'sysadmin', roleName: '系统管理员', legalEntityName: '主品牌', regionScope: '',
}

function renderAt(path: string): string {
  return renderToStaticMarkup(
    <ProfileContext.Provider value={profile}>
      <MemoryRouter initialEntries={[path]}>
        <ProfilePage />
      </MemoryRouter>
    </ProfileContext.Provider>,
  )
}

describe('ProfilePage 分区路由', () => {
  it('security 路由渲染改密表单', () => {
    const html = renderAt('/ucenter/security')
    expect(html).toContain('安全设置')
    expect(html).toContain('旧密码')
    expect(html).toContain('type="password"')
  })

  it('api-keys 路由渲染密钥分区(表头结构)', () => {
    const html = renderAt('/ucenter/api-keys')
    expect(html).toContain('API Key 管理')
    expect(html).toContain('Key 前缀')
    expect(html).toContain('创建 Key')
  })

  it('audit 路由渲染审计分区标题', () => {
    const html = renderAt('/ucenter/audit')
    expect(html).toContain('我的操作审计')
  })

  it('默认路由渲染基本资料表单,且无 email 字段', () => {
    const html = renderAt('/ucenter/profile')
    expect(html).toContain('基本资料')
    expect(html).toContain('13800000000') // phone 初值来自 /auth/me
    expect(html).not.toContain('type="email"')
  })

  it('work 路由渲染快捷入口', () => {
    const html = renderAt('/ucenter/work')
    expect(html).toContain('我的工作')
    expect(html).toContain('我负责的订单')
  })
})
