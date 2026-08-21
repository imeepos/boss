package com.ymm.boss.user.api

import org.json.JSONObject

// 账户组端点封装:仅收 UserApi 未覆盖的新端点,契约见 api/openapi/user.yaml
// POST /auth/reset-password | GET+POST /auth/verify | GET /agreement
object AccountApi {

    // POST /auth/reset-password {phone, smsCode, newPassword}
    suspend fun resetPassword(phone: String, smsCode: String, newPassword: String): JSONObject =
        Api.post("/auth/reset-password", JSONObject()
            .put("phone", phone).put("smsCode", smsCode).put("newPassword", newPassword))

    // GET /auth/verify -> {status,nameMasked,idNoMasked,phoneMasked,latestResult,submitTime,rejectReason,verifyAt,records[]}
    suspend fun verifyStatus(): JSONObject = Api.get("/auth/verify")

    // POST /auth/verify/sms-code:给当前客户绑定手机号发实名验证码(scene=verify,60s 冷却)
    suspend fun sendVerifySmsCode(): JSONObject = Api.post("/auth/verify/sms-code")

    // POST /auth/verify {idType,name,idNo,smsCode,idCardFrontId,idCardBackId} 提交实名核验
    suspend fun submitVerify(idType: String, name: String, idNo: String, smsCode: String,
                             idCardFrontId: Long, idCardBackId: Long): JSONObject =
        Api.post("/auth/verify", JSONObject()
            .put("idType", idType).put("name", name).put("idNo", idNo).put("smsCode", smsCode)
            .put("idCardFrontId", idCardFrontId).put("idCardBackId", idCardBackId))

    // POST /attachments/upload multipart(file) -> {id,fileName,contentType,sizeBytes,...}
    suspend fun uploadAttachment(fileName: String, contentType: String, bytes: ByteArray): JSONObject =
        Api.upload("/attachments/upload", fileName, contentType, bytes)

    // GET /agreement -> {userAgreement[], privacyPolicy[]}
    suspend fun agreement(): JSONObject = Api.get("/agreement")
}
