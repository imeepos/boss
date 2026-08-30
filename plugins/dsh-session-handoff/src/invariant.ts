/**
 * 部署配置的解析与校验：非法配置在插件加载时抛错（fail-closed），不静默吞。
 * @module @ymm/dsh-session-handoff/invariant
 */

export interface HandoffConfig {
  /** 报告落盘目录（相对项目根），自动加入 .git/info/exclude。 */
  outputDir: string
  /** 报告文件名。 */
  fileName: string
  /** 同一项目两次自动盘点的最小间隔（毫秒）；/handoff 命令不受限。 */
  minIntervalMs: number
  /** 主分支名，收尾判定以它为基准。 */
  mainBranch: string
  /** 远端名，推送建议使用。 */
  remote: string
  /** 是否自动把 outputDir 写进 .git/info/exclude。 */
  autoExclude: boolean
}

export const DEFAULT_CONFIG: HandoffConfig = {
  outputDir: '.handoff',
  fileName: 'LATEST.md',
  minIntervalMs: 30_000,
  mainBranch: 'main',
  remote: 'gitea',
  autoExclude: true,
}

/**
 * 校验并填充默认值；类型不符或取值非法即抛 TypeError。
 * @param raw - 行清单或宿主下发的原始配置，允许缺省字段。
 * @returns 一份独立的已校验配置。
 */
export function resolveConfig(raw?: Partial<HandoffConfig>): HandoffConfig {
  const merged = { ...DEFAULT_CONFIG, ...(raw ?? {}) }
  return {
    outputDir: nonEmptyString('outputDir', normalizeDir(merged.outputDir ?? DEFAULT_CONFIG.outputDir)),
    fileName: safeBasename('fileName', merged.fileName ?? DEFAULT_CONFIG.fileName),
    minIntervalMs: nonNegativeNumber('minIntervalMs', merged.minIntervalMs ?? DEFAULT_CONFIG.minIntervalMs),
    mainBranch: nonEmptyString('mainBranch', (merged.mainBranch ?? DEFAULT_CONFIG.mainBranch).trim()),
    remote: nonEmptyString('remote', (merged.remote ?? DEFAULT_CONFIG.remote).trim()),
    autoExclude: trueBoolean('autoExclude', merged.autoExclude ?? DEFAULT_CONFIG.autoExclude),
  }
}

function nonEmptyString(field: string, value: string): string {
  if (typeof value !== 'string' || value.length === 0) {
    throw new TypeError(`session-handoff config.${field} 必须是非空字符串`)
  }
  return value
}

function safeBasename(field: string, value: string): string {
  nonEmptyString(field, value)
  if (value.includes('/') || value.includes('\\')) {
    throw new TypeError(`session-handoff config.${field} 只能是文件名，不能含路径分隔符`)
  }
  return value
}

function nonNegativeNumber(field: string, value: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
    throw new TypeError(`session-handoff config.${field} 必须是 >= 0 的有限数字`)
  }
  return value
}

function trueBoolean(field: string, value: boolean): boolean {
  if (typeof value !== 'boolean') {
    throw new TypeError(`session-handoff config.${field} 必须是布尔值`)
  }
  return value
}

/** 去掉开头的 `./`、`/` 与结尾的 `/`。 */
function normalizeDir(dir: string): string {
  if (typeof dir !== 'string') throw new TypeError('session-handoff config.outputDir 必须是字符串')
  return dir.replace(/^\.?\//, '').replace(/\/+$/, '')
}
