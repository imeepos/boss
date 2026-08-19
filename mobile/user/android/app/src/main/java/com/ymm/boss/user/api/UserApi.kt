package com.ymm.boss.user.api

import org.json.JSONObject

/** 用户端端点封装,按 docs/user/api.js 逐组补齐。 */
object UserApi {

    object auth {
        suspend fun smsCode(phone: String, scene: String): JSONObject =
            Api.post("/auth/sms-code", JSONObject().put("phone", phone).put("scene", scene))

        suspend fun login(phone: String, mode: String, credential: String): JSONObject {
            val body = JSONObject().put("phone", phone).put("mode", mode)
            if (mode == "sms") body.put("smsCode", credential) else body.put("password", credential)
            return Api.post("/auth/login", body)
        }

        suspend fun register(phone: String, smsCode: String, password: String): JSONObject =
            Api.post("/auth/register", JSONObject()
                .put("phone", phone).put("smsCode", smsCode).put("password", password))

        suspend fun logout(): JSONObject = Api.post("/auth/logout")
    }

    object misc {
        suspend fun home(): JSONObject = Api.get("/home")
        suspend fun messages(category: String? = null): JSONObject = Api.get("/messages" + Api.qs(mapOf("category" to category)))
        suspend fun usage(period: String? = null): JSONObject = Api.get("/usage" + Api.qs(mapOf("period" to period)))
    }
}
