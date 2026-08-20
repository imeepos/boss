package com.ymm.boss.user.api

import android.content.Context
import android.content.SharedPreferences

/** 语言偏好明文存储:非敏感配置,无需加密;随 Api.init 初始化。 */
object LangStore {
    private const val LANG_KEY = "boss_user_lang"
    private const val PREFS = "boss_user"

    private lateinit var prefs: SharedPreferences

    fun init(context: Context) {
        prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
    }

    fun read(): String = if (::prefs.isInitialized) prefs.getString(LANG_KEY, "zh") ?: "zh" else "zh"

    fun write(lang: String) {
        if (!::prefs.isInitialized) return
        prefs.edit().putString(LANG_KEY, lang).apply()
    }
}
