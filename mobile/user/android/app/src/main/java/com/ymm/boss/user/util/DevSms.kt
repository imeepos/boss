package com.ymm.boss.user.util

import com.ymm.boss.user.api.DebugApi
import com.ymm.boss.user.api.DevModeStore

/**
 * 开发模式验证码自动回填:调用 IssueSms 成功后,若 DevModeStore 开启,
 * 再调 DebugApi.latestSmsCode 从后台拉取明文,自动写入验证码输入框。
 * release 包/开关关闭 → 直接跳过,无任何网络开销。
 *
 * 返回值:回填成功 → true;关闭或回填失败 → false(调用方继续走正常路径)。
 */
suspend fun devAutoFillSms(phone: String, scene: String, onCode: (String) -> Unit): Boolean {
    if (!DevModeStore.isEnabled()) return false
    return try {
        val r = DebugApi.latestSmsCode(phone, scene)
        val code = r.optString("code")
        if (code.isNotBlank()) { onCode(code); true } else false
    } catch (e: Exception) {
        false
    }
}