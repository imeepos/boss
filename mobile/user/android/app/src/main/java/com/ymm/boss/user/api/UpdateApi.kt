package com.ymm.boss.user.api

import android.content.Context
import android.content.Intent
import android.net.Uri
import com.ymm.boss.user.BuildConfig
import org.json.JSONObject
import java.util.UUID

/**
 * 在线更新检查(契约 fields.md 8F):GET /client/latest 免登录,
 * query versionCode/deviceId;deviceId 本地 UUID 持久化,参与服务端灰度分桶。
 */
object UpdateApi {

    private const val DEVICE_KEY = "boss_device_id"

    /** 设备号:首次生成 UUID 后落 SharedPreferences,灰度命中在设备上稳定。 */
    fun deviceId(context: Context): String {
        val prefs = context.getSharedPreferences("boss_user", Context.MODE_PRIVATE)
        var id = prefs.getString(DEVICE_KEY, "") ?: ""
        if (id.isBlank()) {
            id = UUID.randomUUID().toString()
            prefs.edit().putString(DEVICE_KEY, id).apply()
        }
        return id
    }

    /** 升级判定结果;updateAvailable=false 表示已是最新。 */
    data class UpdateInfo(
        val updateAvailable: Boolean,
        val force: Boolean,
        val version: String,
        val notes: String,
        val sizeMb: String,
        val sha256: String,
        val downloadUrl: String,
    )

    /** 启动/手动检查;网络等异常返回 null(调用方静默或提示稍后再试)。 */
    suspend fun check(context: Context): UpdateInfo? = try {
        parse(Api.get("/client/latest?versionCode=${BuildConfig.VERSION_CODE}&deviceId=${deviceId(context)}"))
    } catch (_: Exception) {
        null
    }

    /** 解析信封 data(独立出来供单测;force=true 时 UI 不提供忽略按钮)。 */
    fun parse(data: JSONObject): UpdateInfo = UpdateInfo(
        updateAvailable = data.optBoolean("updateAvailable"),
        force = data.optBoolean("force"),
        version = data.optString("version"),
        notes = data.optString("notes"),
        sizeMb = formatSize(data.optLong("size")),
        sha256 = data.optString("sha256"),
        downloadUrl = data.optString("downloadUrl"),
    )

    /** APK 绝对下载地址(downloadUrl 为同端相对路径,需拼 deviceId 过灰度门控)。 */
    fun downloadLink(context: Context, info: UpdateInfo): String =
        Api.base.removeSuffix("/") + info.downloadUrl + "?deviceId=" + deviceId(context)

    /** 立即更新:系统浏览器/下载器接管,装包由系统接管(不申请安装权限)。 */
    fun openDownload(context: Context, info: UpdateInfo) {
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(downloadLink(context, info)))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        context.startActivity(intent)
    }

    fun formatSize(bytes: Long): String = when {
        bytes >= 1 shl 20 -> String.format("%.1fMB", bytes / 1048576.0)
        bytes > 0 -> "${bytes / 1024}KB"
        else -> ""
    }
}
