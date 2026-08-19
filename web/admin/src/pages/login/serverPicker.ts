// 服务端选择(登录页用):选择持久化;增删改查由 ServerManagerDialog 承担。
import { activeServerId, listServers, setActiveServerId, type ServerConfig } from '../../lib/serverConfig'

export interface ServerPickerState {
  servers: ServerConfig[]
  activeId: string
}

export function initialPickerState(): ServerPickerState {
  const id = activeServerId()
  if (!id) return { servers: listServers(), activeId: '' }
  return { servers: listServers(), activeId: id }
}

/** 切换服务端并持久化(localStorage 记忆,刷新不丢)。 */
export function pickServer(id: string): string {
  setActiveServerId(id)
  return id
}
