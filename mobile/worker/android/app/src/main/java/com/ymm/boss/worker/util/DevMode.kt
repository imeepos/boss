package com.ymm.boss.worker.util

import android.content.Context
import android.content.SharedPreferences
import com.ymm.boss.worker.BuildConfig

// 开发模式:debug 包默认开启(点击"获取验证码"后调服务端开发端点回拉明文自动回填),
// release 包恒 false——与构建类型绑定,不做 UI 开关(user 端同构)。
class DevMode private constructor(private val prefs: SharedPreferences) {
    var enabled: Boolean
        get() = BuildConfig.DEBUG && prefs.getBoolean(KEY, true)
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
