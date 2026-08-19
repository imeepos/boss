package com.ymm.boss.user.api

import org.json.JSONArray
import org.json.JSONObject

// 产品域端点封装,契约 api/openapi/user/product.yaml。
object ProductApi {

    // GET /products?category= 套餐列表(items + addons)。
    suspend fun list(category: String? = null): JSONObject =
        Api.get("/products" + Api.qs(mapOf("category" to category)))

    // GET /products/{productId} 套餐详情(product + specs + compare)。
    suspend fun detail(productId: String): JSONObject = Api.get("/products/$productId")
}

// 订单域端点封装,契约 api/openapi/user/order.yaml。
object OrderApi {

    // GET /orders?status= 订单列表,status: all/in_progress/done/cancelled。
    suspend fun list(status: String? = null): JSONObject =
        Api.get("/orders" + Api.qs(mapOf("status" to status)))

    // POST /orders 下单(环节1 submitOrder),返回 OrderSummary 含 orderNo。
    suspend fun submit(productId: String, addressId: String): JSONObject =
        Api.post("/orders", JSONObject().put("productId", productId).put("addressId", addressId))

    // GET /orders/{orderNo} 订单详情(含 12 环节 timeline)。
    suspend fun detail(orderNo: String): JSONObject = Api.get("/orders/$orderNo")

    suspend fun cancel(orderNo: String): JSONObject = Api.post("/orders/$orderNo/cancel")

    suspend fun urge(orderNo: String): JSONObject = Api.post("/orders/$orderNo/urge")

    // POST /orders/{orderNo}/change-address 变更安装地址。
    suspend fun changeAddress(orderNo: String, addressId: String): JSONObject =
        Api.post("/orders/$orderNo/change-address", JSONObject().put("addressId", addressId))

    // GET /orders/{orderNo}/rate 待评价订单信息。
    suspend fun rateInfo(orderNo: String): JSONObject = Api.get("/orders/$orderNo/rate")

    // POST /orders/{orderNo}/rate 提交服务评价。
    suspend fun submitRate(
        orderNo: String, stars: Int, attitude: Int, quality: Int, comment: String,
    ): JSONObject = Api.post(
        "/orders/$orderNo/rate",
        JSONObject().put("stars", stars).put("attitude", attitude)
            .put("quality", quality).put("comment", comment),
    )
}

// JSONArray 转 JSONObject 列表,跳过非对象项。
internal fun JSONArray?.toObjList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) out.add(optJSONObject(i) ?: continue)
    return out
}
