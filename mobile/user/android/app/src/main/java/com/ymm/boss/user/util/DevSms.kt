package com.ymm.boss.user.util

import com.ymm.boss.user.BuildConfig
import com.ymm.boss.user.api.DebugApi

/**
 * 开发模式验证码自动回填:debug 包发码成功后,直接调后台拉明文回填输入框。
 * release 包 → BuildConfig.DEBUG=false → 跳过,无任何网络开销。
 * 后端未开 BOSS_DEBUG_SMS → 404 → catch 后静默跳过,不报错。
 */
suspend fun devAutoFillSms(phone: String, scene: String, onCode: (String) -> Unit): Boolean {
    if (!BuildConfig.DEBUG) return false
    return try {
        val r = DebugApi.latestSmsCode(phone, scene)
        val code = r.optString("code")
        if (code.isNotBlank()) { onCode(code); true } else false
    } catch (_: Exception) {
        false
    }
}