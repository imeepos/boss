package com.ymm.boss.user.api

import org.json.JSONArray
import org.json.JSONObject

/**
 * 用户端积分域封装,契约 api/openapi/user/loy.yaml(LOGO 000119)。
 * 读为主:概览/等级/任务/可兑换模板;兑为辅:/points/exchange 先扣后发(发券失败后端补偿回补)。
 * 禁止改动 UserApi.kt 与 Api.kt,新接口一律走这里。
 */
object PointsApi {

    /** 我的积分:data.balance + data.entries(近 100 条流水)。 */
    suspend fun overview(): JSONObject = Api.get("/points")

    /** 我的积分等级:data.tier 可为 null(未达任何档)。 */
    suspend fun tier(): JSONObject = Api.get("/points/tier")

    /** 积分任务列表:data.tasks,completedAt 非空=本周期已完成。 */
    suspend fun tasks(): JSONArray = Api.get("/points/tasks").optJSONArray("tasks") ?: JSONArray()

    /** 完成任务领积分(周期内幂等,重复调后端返回冲突)。返回 data.balance。 */
    suspend fun completeTask(taskId: Long): JSONObject = Api.post("/points/tasks/$taskId/complete")

    /** 积分可兑换券模板列表:data.items(templateId/name/type/faceValue/threshold/pointsPrice)。 */
    suspend fun exchangeOffers(): JSONArray =
        Api.get("/points/exchange-offers").optJSONArray("items") ?: JSONArray()

    /** 积分换券(先扣后发,发券失败后端补偿回补)。返回 data.couponId/cost。 */
    suspend fun exchange(templateId: Long): JSONObject =
        Api.post("/points/exchange", JSONObject().put("templateId", templateId))
}