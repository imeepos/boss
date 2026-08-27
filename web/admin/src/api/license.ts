// 系统授权接口:对接 /license/status 与 /license/activate(release-platform 离线授权)。
import { apiFetch } from './client'

export interface LicenseStatus {
  activated: boolean
  enabled: boolean
  reason?: string
  licenseId?: string
  productId?: string
  licenseType?: string
  deviceId?: string
  fingerprint?: string
  expiresAt?: string
  graceEndsAt?: string
  inGrace?: boolean
  checkedAt?: string
}

/** 查询授权状态(HTTP 恒 200 envelope;data 内 activated 区分)。 */
export function fetchLicenseStatus(): Promise<LicenseStatus | null> {
  return apiFetch<LicenseStatus>('/license/status')
}

/** 激活码兑换:调 release-platform 兑码并本地落盘证书。 */
export function activateLicense(activationCode: string): Promise<LicenseStatus | null> {
  return apiFetch<LicenseStatus>('/license/activate', {
    method: 'POST',
    body: { activationCode },
  })
}