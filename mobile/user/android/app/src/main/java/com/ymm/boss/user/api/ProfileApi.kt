package com.ymm.boss.user.api

import org.json.JSONArray
import org.json.JSONObject

// * 个人中心组端点封装,契约 api/openapi/user.yaml(profile/misc 引用)。
// * 禁止改动 UserApi.kt 与 Api.kt,新页面一律走这里。
object ProfileApi {

    suspend fun get(): JSONObject = Api.get("/profile")

    suspend fun security(): JSONObject = Api.get("/profile/security")

    suspend fun changePassword(oldPassword: String, newPassword: String): JSONObject =
        Api.put("/profile/security/password", JSONObject()
            .put("oldPassword", oldPassword).put("newPassword", newPassword))

    suspend fun changePhone(newPhone: String, smsCode: String): JSONObject =
        Api.put("/profile/security/phone", JSONObject()
            .put("newPhone", newPhone).put("smsCode", smsCode))

    suspend fun notifySettings(): JSONObject = Api.get("/profile/notify-settings")

    suspend fun saveNotifySettings(payload: JSONObject): JSONObject =
        Api.put("/profile/notify-settings", payload)

    suspend fun setLanguage(language: String): JSONObject =
        Api.put("/profile/language", JSONObject().put("language", language))

    /** 分页拉取家庭地址(契约 misc.yaml /addresses,后端返回 page/pageSize/total/hasMore)。 */
    suspend fun addresses(page: Int = 1, pageSize: Int = 20): JSONObject =
        Api.get("/addresses" + Api.qs(
            mapOf("page" to page.toString(), "pageSize" to pageSize.toString())
        ))

    suspend fun createAddress(payload: JSONObject): JSONObject = Api.post("/addresses", payload)

    suspend fun updateAddress(addressId: String, payload: JSONObject): JSONObject =
        Api.put("/addresses/$addressId", payload)

    /** 删除家庭地址;后端 404 → Api.HttpError(40400),页面据此区分"已被删除/已撤回"。 */
    suspend fun deleteAddress(addressId: String): JSONObject =
        Api.delete("/addresses/$addressId")

    /** 地址层级树子节点(parentId=0 顶层),级联选择逐层懒加载;契约 misc.yaml /address-tree。 */
    suspend fun addressTree(parentId: Long = 0): List<JSONObject> =
        Api.get("/address-tree?parentId=$parentId").optJSONArray("items").toObjectList()

    /** 地址全树搜索(命中+祖先链);q 空串会被后端 42200 拒绝,调用方自行拦截。 */
    suspend fun addressTreeSearch(q: String): List<JSONObject> =
        Api.get("/address-tree/search" + Api.qs(mapOf("q" to q))).optJSONArray("items").toObjectList()

    suspend fun readAllMessages(): JSONObject = Api.post("/messages/read-all")

    /**
     * 单条消息标为已读。404 表示消息不存在或不属于当前用户(乐观更新可据此回滚)。
     */
    suspend fun readMessage(messageId: String): JSONObject =
        Api.put("/messages/$messageId/read", JSONObject())
}

// JSONArray 转 JSONObject 列表,跳过非法项;页面侧通用工具。
fun JSONArray?.toObjectList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) out.add(optJSONObject(i) ?: continue)
    return out
}
