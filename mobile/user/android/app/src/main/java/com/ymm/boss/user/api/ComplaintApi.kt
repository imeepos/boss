package com.ymm.boss.user.api

import org.json.JSONObject

/**
 * 投诉与建议端点,契约 api/openapi/user/customer-service.yaml。
 * 列表分页 + 详情含处理历史 timeline;提交 description/contact/relOrderNo 真实落库(迁移 000151)。
 */
object ComplaintApi {

    /**
     * GET /complaints?page=&pageSize= 我的投诉分页列表;
     * 返回 items + page + pageSize + hasMore。
     */
    suspend fun list(page: Int = 1, pageSize: Int = 10): JSONObject =
        Api.get("/complaints" + Api.qs(mapOf("page" to page.toString(), "pageSize" to pageSize.toString())))

    /**
     * POST /complaints 提交投诉/建议;type 取值 attitude/quality/billing/suggestion/other;
     * description 必填 4-500 字符,contact/relOrderNo 可选。
     */
    suspend fun submit(
        type: String,
        description: String,
        contact: String? = null,
        relOrderNo: String? = null,
    ): JSONObject =
        Api.post(
            "/complaints",
            JSONObject().put("type", type).put("description", description)
                .apply {
                    if (!contact.isNullOrBlank()) put("contact", contact)
                    if (!relOrderNo.isNullOrBlank()) put("relOrderNo", relOrderNo)
                },
        )

    /**
     * GET /complaints/{ticketNo} 投诉详情,含 {complaint, timeline};
     * timeline[0] 固定为提交投诉节点(DONE),其后为 cs_ticket_events 派生。
     */
    suspend fun detail(ticketNo: String): JSONObject = Api.get("/complaints/$ticketNo")
}