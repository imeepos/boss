package com.ymm.boss.user.api

import android.content.Context
import org.json.JSONObject

/**
 * 推送设备注册(契约 user/misc.yaml POST /push/device,后端 push_devices 000095 PG 持久化)。
 * 幂等:同 registrationId 服务端 UPSERT 换绑主体(换账号登录同一设备自动 rebinding)。
 * 目前无推送 SDK,以设备持久 UUID 作 registrationId、vendor=app_track 自标记;
 * 未来接入 JPush 等真实通道后,同处换真实 RegistrationID 即可(通知权限 POST_NOTIFICATIONS 已立项)。
 * 所有失败静默(后台尽力而为,不打断业务流),Log 留痕便于排查。
 */
object PushApi {

    private const val VENDOR_TRACK = "app_track"

    /** 上报设备注册;失败返回 false(调用方无需处理)。 */
    suspend fun register(registrationId: String, vendor: String = VENDOR_TRACK): Boolean = try {
        val r = Api.post("/push/device", JSONObject()
            .put("registrationId", registrationId)
            .put("vendor", vendor))
        r.optBoolean("ok", false)
    } catch (e: Exception) {
        android.util.Log.d("PushApi", "register FAILED: ${e.message ?: e}")
        false
    }

    /**
     * 以设备持久 UUID(与 UpdateApi 分桶同源 boss_device_id)注册。
     * 服务端 validRegistrationID 仅收 [0-9a-zA-Z](JPush 形态),去连字符规范化(仍唯一)。
     */
    suspend fun registerDevice(context: Context): Boolean =
        register(UpdateApi.deviceId(context.applicationContext).replace("-", ""))
}