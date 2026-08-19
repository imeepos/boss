package com.ymm.boss.user.api

import org.json.JSONObject

/** 客服/帮助域端点,契约见 api/openapi/user/customer-service.yaml 与 misc.yaml。 */
object ServiceApi {
    suspend fun chat(message: String): JSONObject =
        Api.post("/service/chat", JSONObject().put("message", message))

    suspend fun faq(): JSONObject = Api.get("/service/faq")
    suspend fun diySteps(): JSONObject = Api.get("/diy/steps")
}
