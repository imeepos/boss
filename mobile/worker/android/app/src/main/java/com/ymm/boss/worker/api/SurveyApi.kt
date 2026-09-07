package com.ymm.boss.worker.api

import org.json.JSONArray
import org.json.JSONObject

// 勘测任务(契约 api/openapi/worker/survey.yaml,服务端 W7 000223)。
object SurveyApi {
    suspend fun list(status: String? = null): JSONObject =
        Api.get("/surveys" + qs(mapOf("status" to status)))

    suspend fun detail(id: Long): JSONObject = Api.get("/surveys/$id")
    suspend fun accept(id: Long): JSONObject = Api.post("/surveys/$id/accept")

    // 现场回填(append-only):clientMsgId 幂等,弱网重传不重复落行。
    suspend fun report(
        id: Long, lat: Double, lng: Double, facilityNote: String,
        suggestion: String, photoIds: List<Long>, clientMsgId: String
    ): JSONObject = Api.post("/surveys/$id/reports", JSONObject()
        .put("lat", lat).put("lng", lng)
        .put("facilityNote", facilityNote).put("suggestion", suggestion)
        .put("photoIds", JSONArray(photoIds)).put("clientMsgId", clientMsgId))

    // 照片经通用附件域上传(上传者身份=当前师傅),返回 attachment id 供回填引用。
    suspend fun uploadPhoto(fileName: String, contentType: String, bytes: ByteArray): JSONObject =
        Api.upload("/attachments/upload", fileName, contentType, bytes)
}

// 施工进度上报(契约 api/openapi/worker/construction.yaml,服务端 W7 000224,F5a)。
object ConstructionApi {
    suspend fun list(): JSONObject = Api.get("/constructions")
    suspend fun detail(id: Long): JSONObject = Api.get("/constructions/$id")

    suspend fun progress(
        id: Long, facilityCode: String, doneQty: Double, lat: Double, lng: Double,
        note: String, photoIds: List<Long>, clientMsgId: String
    ): JSONObject = Api.post("/constructions/$id/progress", JSONObject()
        .put("facilityCode", facilityCode).put("doneQty", doneQty)
        .put("lat", lat).put("lng", lng).put("note", note)
        .put("photoIds", JSONArray(photoIds)).put("clientMsgId", clientMsgId))
}