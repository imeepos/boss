package com.ymm.boss.user.api

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Token 加密存储:EncryptedSharedPreferences 优先,创建失败(如设备无密钥库)回落明文 prefs。
 * 首次初始化时把旧明文 token 迁移进加密存储并清除旧值。
 */
object TokenStore {
    private const val TOKEN_KEY = "boss_user_token"
    private const val SECURE_FILE = "boss_user_secure"
    private const val PLAIN_PREFS = "boss_user"

    private lateinit var prefs: SharedPreferences

    fun init(context: Context) {
        val appCtx = context.applicationContext
        prefs = encrypted(appCtx) ?: appCtx.getSharedPreferences(PLAIN_PREFS, Context.MODE_PRIVATE)
        migratePlainToken(appCtx)
    }

    private fun encrypted(context: Context): SharedPreferences? = try {
        val master = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context, SECURE_FILE, master,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    } catch (e: Exception) {
        null
    }

    private fun migratePlainToken(context: Context) {
        val plain = context.getSharedPreferences(PLAIN_PREFS, Context.MODE_PRIVATE)
        val legacy = plain.getString(TOKEN_KEY, null) ?: return
        if (prefs.getString(TOKEN_KEY, null).isNullOrBlank() && legacy.isNotBlank()) {
            prefs.edit().putString(TOKEN_KEY, legacy).apply()
        }
        plain.edit().remove(TOKEN_KEY).apply()
    }

    fun read(): String = if (::prefs.isInitialized) prefs.getString(TOKEN_KEY, "") ?: "" else ""

    fun write(token: String?) {
        if (!::prefs.isInitialized) return
        prefs.edit().apply {
            if (token.isNullOrBlank()) remove(TOKEN_KEY) else putString(TOKEN_KEY, token)
        }.apply()
    }
}
