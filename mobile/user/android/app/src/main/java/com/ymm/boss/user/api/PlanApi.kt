package com.ymm.boss.user.api

import org.json.JSONObject

// 套餐组端点封装,契约: api/openapi/user/{profile,product,order}.yaml
object PlanApi {

    // GET /profile -> Profile.plan(Plan: planId/name/monthlyFee/contractEnd/status/installAddress)
    suspend fun profile(): JSONObject = Api.get("/profile")

    // GET /products -> { items: Product[] }(ChangePage 选新套餐用)
    suspend fun products(category: String? = null): JSONObject =
        Api.get("/products" + Api.qs(mapOf("category" to category)))

    // POST /plans/{planId}/change {targetPlanId,effectiveMode,staticIp} -> OrderSummary
    suspend fun change(planId: String, targetPlanId: String, effectiveMode: String, staticIp: Boolean): JSONObject =
        Api.post("/plans/$planId/change", JSONObject()
            .put("targetPlanId", targetPlanId)
            .put("effectiveMode", effectiveMode)
            .put("staticIp", staticIp))

    // POST /plans/{planId}/move {community,building,door,expectDate} -> OrderSummary
    suspend fun move(planId: String, community: String, building: String, door: String, expectDate: String): JSONObject =
        Api.post("/plans/$planId/move", JSONObject()
            .put("community", community)
            .put("building", building)
            .put("door", door)
            .put("expectDate", expectDate))

    // GET /plans/{planId}/cancel -> CancelPreview{unpaidBills[],penalty,penaltyDesc}
    suspend fun cancelPreview(planId: String): JSONObject = Api.get("/plans/$planId/cancel")

    // POST /plans/{planId}/cancel {reason} -> OrderSummary
    suspend fun cancel(planId: String, reason: String): JSONObject =
        Api.post("/plans/$planId/cancel", JSONObject().put("reason", reason))

    // GET /addons -> { available: Addon[], subscribed: Addon[] }
    suspend fun addons(): JSONObject = Api.get("/addons")

    // POST /addons/{addonId}/subscribe -> Ok
    suspend fun subscribe(addonId: String): JSONObject = Api.post("/addons/$addonId/subscribe")

    // POST /addons/{addonId}/unsubscribe -> Ok
    suspend fun unsubscribe(addonId: String): JSONObject = Api.post("/addons/$addonId/unsubscribe")
}
