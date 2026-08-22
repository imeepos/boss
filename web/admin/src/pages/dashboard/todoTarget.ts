// 待办跳转目标:source(后端文案,见 internal/httpapi/admin/dashboard.go dashboardTodos)
// → 菜单页 key → 注册路由 path。经 PAGE_BY_KEY 解析,禁止写死 path(曾写死 /alarm 而
// 注册路由是 /alarm/alarm,点击待办 404)。
import { PAGE_BY_KEY } from '../../router/menu.def'

const SOURCE_TO_KEY: Record<string, string> = {
  '派单池': 'dispatch',
  '告警中心': 'alarm',
}

/** 待办 source → 路由 path;未知 source 返回 null(不跳转)。 */
export function todoTarget(source: string): string | null {
  const key = SOURCE_TO_KEY[source]
  if (!key) return null
  return PAGE_BY_KEY.get(key)?.item.path ?? null
}
