package com.ymm.boss.worker.util

import android.content.Context
import android.content.SharedPreferences

// 开发模式开关:开启时点击"获取验证码"会向服务端开发端点回拉验证码并自动回填输入框。
// 持久化到 SharedPreferences,与登录态无关;主开关目的是联调时不读真实短信方便测试。
class DevMode private constructor(private val prefs: SharedPreferences) {
    var enabled: Boolean
        get() = prefs.getBoolean(KEY, false)
        set(v) { prefs.edit().putBoolean(KEY, v).apply() }

    companion object {
        private const val KEY = "boss_worker_dev_mode"
        private const val PREFS = "boss_worker"
        @Volatile private var instance: DevMode? = null

        fun get(ctx: Context): DevMode = instance ?: synchronized(this) {
            instance ?: DevMode(
                ctx.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            ).also { instance = it }
        }
    }
}
