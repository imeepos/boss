package com.ymm.boss.user.api

import org.json.JSONObject

// 报障/投诉域端点封装,契约 api/openapi/user/customer-service.yaml。

// faults 端点:列表/提交/详情(报障 6 环节时间轴)。
object FaultApi {

    // GET /faults 报修记录列表(items: Fault)。
    suspend fun list(): JSONObject = Api.get("/faults")

    // POST /faults 提交报修,faultType: no_internet/slow/ont_fault/other,返回 Fault 含 ticketNo。
    suspend fun submit(faultType: String, address: String, description: String, contact: String): JSONObject =
        Api.post(
            "/faults",
            JSONObject().put("faultType", faultType).put("address", address)
                .put("description", description).put("contact", contact),
        )

    // GET /faults/{ticketNo} 报修详情(fault + technician + timeline)。
    suspend fun detail(ticketNo: String): JSONObject = Api.get("/faults/$ticketNo")
}

// complaints 端点:我的投诉列表/提交投诉建议。
object ComplaintApi {

    // GET /complaints 我的投诉列表(items: Complaint)。
    suspend fun list(): JSONObject = Api.get("/complaints")

    // POST /complaints 提交投诉/建议,type: attitude/quality/billing/suggestion/other。
    suspend fun submit(type: String, relOrderNo: String, description: String, contact: String): JSONObject =
        Api.post(
            "/complaints",
            JSONObject().put("type", type).put("relOrderNo", relOrderNo)
                .put("description", description).put("contact", contact),
        )
}
