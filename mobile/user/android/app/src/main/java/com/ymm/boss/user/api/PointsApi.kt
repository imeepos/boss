package com.ymm.boss.user.api

import org.json.JSONArray
import org.json.JSONObject

/**
 * 用户端积分域封装,契约 api/openapi/user/loy.yaml(LOGO 000119)。
 * 读为主:概览/等级/任务;兑为辅:/points/exchange 需兑换券模板(后端暂未提供列表端点,页面占位)。
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

    /** 积分换券(契约端点;templateId 需模板列表,后端就绪前页面不走此路径)。 */
    suspend fun exchange(templateId: Long): JSONObject =
        Api.post("/points/exchange", JSONObject().put("templateId", templateId))
}