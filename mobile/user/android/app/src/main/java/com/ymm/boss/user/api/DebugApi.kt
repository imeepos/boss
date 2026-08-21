package com.ymm.boss.user.api

import org.json.JSONObject

/**
 * 开发模式专用端点:开发模式下"获取验证码"后,App 调此接口从后台拉取明文回填。
 * 服务端对应 GET /api/user/v1/debug/sms-code,仅当 DebugSmsEnabled=true 时挂载;
 * release 包前端永远走不通(BuildConfig.DEBUG=false → DevModeStore.isEnabled=false)。
 */
object DebugApi {
    /**
     * GET /debug/sms-code?phone=&scene= -> {code,issuedAt}
     * verify 场景 phone 可空(服务端从 JWT 推导);其他场景 phone 必填。
     */
    suspend fun latestSmsCode(phone: String, scene: String): JSONObject =
        Api.get("/debug/sms-code" + Api.qs(mapOf("phone" to phone.ifBlank { null }, "scene" to scene)))
}