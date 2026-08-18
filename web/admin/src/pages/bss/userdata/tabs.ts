// 用户端配置页 Tab 定义:路径与动作对齐 internal/app/http_userdata_more.go 实际路由。
export interface Row {
  id: number | string
  name?: string
  title?: string
  question?: string
  code?: string
  label?: string
  status?: string
  enabled?: boolean
  [k: string]: unknown
}

export interface TabDef {
  key: string
  path: string
  action?: string
  actionPath?: string
}

export const TABS: TabDef[] = [
  { key: 'notify', path: '/user-notify-settings' },
  { key: 'addons', path: '/addons', action: 'toggle', actionPath: '/addons' },
  { key: 'coupons', path: '/coupons', action: 'disable', actionPath: '/coupons' },
  { key: 'topup', path: '/topup-denominations' },
  { key: 'faqs', path: '/user-faqs', action: 'toggle', actionPath: '/user-faqs' },
  { key: 'guides', path: '/diy-guides', action: 'toggle', actionPath: '/diy-guides' },
  { key: 'invite', path: '/invite-config' },
]
