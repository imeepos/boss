package com.ymm.boss.user.api

import android.content.Context
import android.content.SharedPreferences

/**
 * 开发模式开关:开启后,"获取验证码"会在后端签发后立即调 DebugApi 拉取明文回填。
 * 仅 debug 包生效:release 包 BuildConfig.DEBUG=false → 永久关闭,不会出现在 UI 也不写入存储。
 * 存储在明文 prefs(非敏感配置,与 LangStore 同源)。
 */
object DevModeStore {
    private const val KEY = "boss_user_dev_mode"
    private const val PREFS = "boss_user_dev"

    private lateinit var prefs: SharedPreferences

    fun init(context: Context) {
        prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
    }

    /** release 包强制 false;debug 包读 prefs,默认 false。 */
    fun isEnabled(): Boolean {
        if (!com.ymm.boss.user.BuildConfig.DEBUG) return false
        if (!::prefs.isInitialized) return false
        return prefs.getBoolean(KEY, false)
    }

    fun setEnabled(enabled: Boolean) {
        if (!::prefs.isInitialized) return
        prefs.edit().putBoolean(KEY, enabled).apply()
    }
}