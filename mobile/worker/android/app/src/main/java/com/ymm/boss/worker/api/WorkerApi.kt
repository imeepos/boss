package com.ymm.boss.worker.api

import org.json.JSONArray
import org.json.JSONObject

// 工单号 URL query 拼接工具
private fun qs(params: Map<String, String?>): String {
    val s = StringBuilder()
    for ((k, v) in params) {
        if (v.isNullOrEmpty()) continue
        s.append(if (s.isEmpty()) '?' else '&').append(k).append('=')
            .append(java.net.URLEncoder.encode(v, "UTF-8"))
    }
    return s.toString()
}

object AuthApi {
    suspend fun smsCode(phone: String): JSONObject =
        Api.post("/auth/sms-code", JSONObject().put("phone", phone))

    suspend fun login(phone: String, mode: String, credential: String): JSONObject {
        val body = JSONObject().put("phone", phone).put("mode", mode)
        if (mode == "sms") body.put("smsCode", credential) else body.put("password", credential)
        return Api.post("/auth/login", body)
    }

    suspend fun logout(): JSONObject = Api.post("/auth/logout")
}

object HomeApi {
    suspend fun get(): JSONObject = Api.get("/home")
}

object TicketApi {
    suspend fun list(status: String? = null): JSONObject =
        Api.get("/tickets" + qs(mapOf("status" to status)))

    suspend fun history(period: String? = null): JSONObject =
        Api.get("/tickets/history" + qs(mapOf("period" to period)))

    suspend fun detail(no: String): JSONObject = Api.get("/tickets/$no")
    suspend fun accept(no: String): JSONObject = Api.post("/tickets/$no/accept")
    suspend fun checkin(no: String, lat: Double?, lng: Double?): JSONObject =
        Api.post("/tickets/$no/checkin", JSONObject().put("lat", lat ?: 0).put("lng", lng ?: 0))

    suspend fun navi(no: String): JSONObject = Api.get("/tickets/$no/navi")

    suspend fun transfer(no: String, reason: String, targetId: Long?, remark: String): JSONObject =
        Api.post("/tickets/$no/transfer", JSONObject().put("reason", reason)
            .put("targetWorkerId", targetId).put("remark", remark))

    suspend fun reschedule(no: String, date: String, slot: String, reason: String, remark: String): JSONObject =
        Api.post("/tickets/$no/reschedule", JSONObject().put("newDate", date)
            .put("newSlot", slot).put("reason", reason).put("remark", remark))

    suspend fun rollback(no: String): JSONObject = Api.post("/tickets/$no/rollback")
    suspend fun retry(no: String): JSONObject = Api.post("/tickets/$no/retry")

    suspend fun complaint(no: String, category: String, content: String): JSONObject =
        Api.post("/tickets/$no/complaint", JSONObject().put("category", category).put("content", content))

    suspend fun repairReport(no: String, result: String, remark: String): JSONObject =
        Api.post("/tickets/$no/repair-report", JSONObject().put("result", result).put("remark", remark))
}

object HallApi {
    suspend fun list(): JSONObject = Api.get("/hall")
    suspend fun grab(no: String): JSONObject = Api.post("/hall/$no/grab")
}

object ScanApi {
    suspend fun bind(no: String, epc: String, offline: Boolean): JSONObject =
        Api.post("/tickets/$no/scan-bind", JSONObject().put("epc", epc).put("offline", offline))

    suspend fun abnormal(no: String, payload: JSONObject): JSONObject =
        Api.post("/tickets/$no/scan-abnormal", payload)

    suspend fun photos(no: String): JSONObject = Api.get("/tickets/$no/photos")
    suspend fun uploadPhoto(no: String, scene: String): JSONObject =
        Api.post("/tickets/$no/photos", JSONObject().put("scene", scene))

    suspend fun report(no: String): JSONObject = Api.get("/tickets/$no/report")
    suspend fun submitReport(no: String, remark: String): JSONObject =
        Api.post("/tickets/$no/report", JSONObject().put("remark", remark))

    suspend fun activation(no: String): JSONObject = Api.get("/tickets/$no/activation")
    suspend fun activate(no: String): JSONObject = Api.post("/tickets/$no/activate")

    suspend fun sign(no: String, signatureData: String): JSONObject =
        Api.post("/tickets/$no/sign", JSONObject().put("signatureData", signatureData))

    suspend fun charge(no: String): JSONObject = Api.get("/tickets/$no/charge")

    suspend fun submitCharge(no: String, amount: Double, payMethod: String): JSONObject =
        Api.post("/tickets/$no/charge", JSONObject().put("amount", amount).put("payMethod", payMethod))
}

object AssetApi {
    suspend fun dismantleScan(no: String, epc: String): JSONObject =
        Api.post("/tickets/$no/dismantle/scan", JSONObject().put("epc", epc))

    suspend fun replace(no: String): JSONObject = Api.get("/tickets/$no/replace")

    suspend fun submitReplace(no: String, oldEpc: String, newEpc: String): JSONObject =
        Api.post("/tickets/$no/replace", JSONObject().put("oldEpc", oldEpc).put("newEpc", newEpc))

    suspend fun returnAsset(epc: String): JSONObject = Api.post("/assets/$epc/return")
    suspend fun materials(): JSONObject = Api.get("/materials")
    suspend fun materialOut(id: String): JSONObject = Api.post("/materials/$id/out")
    suspend fun tools(): JSONObject = Api.get("/materials/tools")
    suspend fun borrowTool(id: String): JSONObject = Api.post("/materials/tools/$id/borrow")
    suspend fun giveBackTool(id: String): JSONObject = Api.post("/materials/tools/$id/give-back")
    suspend fun maintenance(): JSONObject = Api.get("/maintenance")
    suspend fun measure(no: String): JSONObject = Api.get("/tickets/$no/measure")
    suspend fun resources(no: String): JSONObject = Api.get("/tickets/$no/resources")
}

object ProfileApi {
    suspend fun get(): JSONObject = Api.get("/profile")
    suspend fun performance(period: String? = null): JSONObject =
        Api.get("/performance" + qs(mapOf("period" to period)))

    suspend fun schedule(month: String? = null): JSONObject =
        Api.get("/schedule" + qs(mapOf("month" to month)))

    suspend fun clock(type: String): JSONObject =
        Api.post("/schedule/clock", JSONObject().put("type", type))

    suspend fun settings(): JSONObject = Api.get("/settings")
    suspend fun saveSettings(s: JSONObject): JSONObject = Api.put("/settings", s)
    suspend fun feedbacks(): JSONObject = Api.get("/feedbacks")
}

object MiscApi {
    suspend fun messages(): JSONObject = Api.get("/messages")
    suspend fun readAll(): JSONObject = Api.post("/messages/read-all")
    suspend fun clear(): JSONObject = Api.post("/messages/clear")
    suspend fun notices(): JSONObject = Api.get("/notices")
    suspend fun faq(keyword: String? = null): JSONObject =
        Api.get("/help/faq" + qs(mapOf("keyword" to keyword)))

    suspend fun serviceMessages(): JSONObject = Api.get("/service/messages")

    suspend fun sendServiceMessage(content: String): JSONObject =
        Api.post("/service/messages", JSONObject().put("content", content))

    suspend fun safetyCheck(workType: String, checklist: JSONArray): JSONObject =
        Api.post("/safety/checks", JSONObject().put("workType", workType).put("checklist", checklist))
}
