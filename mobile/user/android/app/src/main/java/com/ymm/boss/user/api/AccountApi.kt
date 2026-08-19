package com.ymm.boss.user.api

import org.json.JSONObject

// 账户组端点封装:仅收 UserApi 未覆盖的新端点,契约见 api/openapi/user.yaml
// POST /auth/reset-password | GET+POST /auth/verify | GET /agreement
object AccountApi {

    // POST /auth/reset-password {phone, smsCode, newPassword}
    suspend fun resetPassword(phone: String, smsCode: String, newPassword: String): JSONObject =
        Api.post("/auth/reset-password", JSONObject()
            .put("phone", phone).put("smsCode", smsCode).put("newPassword", newPassword))

    // GET /auth/verify -> {status, nameMasked, idNoMasked, verifyAt, records[]}
    suspend fun verifyStatus(): JSONObject = Api.get("/auth/verify")

    // POST /auth/verify {idType, name, idNo} 发起(重新)实名核验
    suspend fun submitVerify(idType: String, name: String, idNo: String): JSONObject =
        Api.post("/auth/verify", JSONObject()
            .put("idType", idType).put("name", name).put("idNo", idNo))

    // GET /agreement -> {userAgreement[], privacyPolicy[]}
    suspend fun agreement(): JSONObject = Api.get("/agreement")
}
