// 认证日志失败原因文案:契约枚举 docs/contract/fields.md §8A(fail_reason,空=SUCCESS 或存量行)。
import type { Translations } from '../../i18n/types'

type AaaLogTexts = Translations['pages']['aaaLogPage']

// fail_reason 枚举 → i18n 键;未知码原样透出留痕(新增枚举未翻案时可见原始码)。
const FAIL_REASON_KEYS: Record<string, keyof AaaLogTexts> = {
  BAD_CREDENTIAL: 'failBadCredential',
  LOCKED: 'failLocked',
  NOT_FOUND: 'failNotFound',
  SUSPENDED: 'failSuspended',
  CLOSED: 'failClosed',
  CONCURRENT_LIMIT: 'failConcurrent',
}

/** 认证日志行失败原因列文案:成功行=成功;失败无码(存量)=不适用;未知码透出原码。 */
export function failReasonText(row: { result: string; failReason?: string }, a: AaaLogTexts): string {
  if (row.result === 'SUCCESS') return a.failReasonOK
  const code = row.failReason ?? ''
  if (!code) return a.failReasonNA
  const key = FAIL_REASON_KEYS[code]
  return key ? (a[key] as string) : code
}
