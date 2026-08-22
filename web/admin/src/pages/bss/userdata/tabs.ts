// 用户端配置页 Tab 定义:路径与动作对齐 internal/app/http_userdata_more.go 实际路由。
// 主键字段名按后端 SQL 别名一一对应,见 internal/domain/customer/userdata/pg_lists.go。
export interface Row {
  [k: string]: unknown
}

export interface TabDef {
  key: string
  path: string
  /** 行级主键字段名(后端 SQL AS 出来的列名),渲染 ID 列与拼动作 URL 都用它。 */
  idKey: string
  action?: string
  actionPath?: string
}

export const TABS: TabDef[] = [
  { key: 'notify', path: '/user-notify-settings', idKey: 'customerId' },
  { key: 'addons', path: '/addons', idKey: 'addonId', action: 'toggle', actionPath: '/addons' },
  { key: 'coupons', path: '/coupons', idKey: 'couponId', action: 'disable', actionPath: '/coupons' },
  { key: 'topup', path: '/topup-denominations', idKey: 'denomId' },
  { key: 'faqs', path: '/user-faqs', idKey: 'faqId', action: 'toggle', actionPath: '/user-faqs' },
  { key: 'guides', path: '/diy-guides', idKey: 'guideId', action: 'toggle', actionPath: '/diy-guides' },
  { key: 'invite', path: '/invite-config', idKey: 'id' },
]