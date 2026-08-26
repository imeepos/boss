package com.ymm.boss.user.api

import org.json.JSONObject

// 报障端点封装,契约 api/openapi/user/customer-service.yaml。
// 投诉与建议拆见 ComplaintApi.kt。

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

    // POST /faults/{ticketNo}/urge 催单,返回 {ok, urgedAt}。
    suspend fun urge(ticketNo: String): JSONObject = Api.post("/faults/$ticketNo/urge")

    // GET /faults/{ticketNo}/contact 师傅明文联系方式 {technicianName, technicianPhone},未指派时 404。
    suspend fun contact(ticketNo: String): JSONObject = Api.get("/faults/$ticketNo/contact")
}
