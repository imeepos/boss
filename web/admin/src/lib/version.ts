// 版本漂移判定:构建注入的 commit vs 服务端 /healthz 自报 commit(版本自证消费端)。
// 纯函数独立成模块,便于 node 环境单测。
import { activeServer, normalizeBaseUrl } from './serverConfig'

/**
 * 判定前端是否落后于服务端:双侧 commit 均非空且不一致。
 * 任一侧为空(未配置服务/旧服务端无字段/本地 dev)不算漂移,避免误报。
 */
export function isVersionDrift(buildCommit: string, serverCommit: string): boolean {
  return !!buildCommit && !!serverCommit && buildCommit !== serverCommit
}

/** 当前生效服务端的 /healthz 地址;未配置服务时返回 null(不发请求)。 */
export function healthzUrl(): string | null {
  const base = activeServer()?.baseUrl
  return base ? `${normalizeBaseUrl(base)}/healthz` : null
}

/** 角标提示键:记录用户已对哪个服务端 commit 点过刷新(每次部署只提示一次)。 */
export const VERSION_SEEN_KEY = 'boss.version.seen'

/**
 * 是否应提示:存在漂移 且 用户尚未对该服务端 commit 提示过。
 * server-only 部署时 web 内容不变,刷新后 commit 仍不匹配——
 * 无 seen 记忆角标会永久驻留成噪音(b2a1d950 上线当日实测推演)。
 */
export function shouldShowVersionBadge(
  buildCommit: string,
  serverCommit: string,
  seenCommit: string,
): boolean {
  return isVersionDrift(buildCommit, serverCommit) && seenCommit !== serverCommit
}
