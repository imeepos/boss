package com.ymm.boss.user.api

import org.json.JSONArray
import org.json.JSONObject

/**
 * 账单费用组接口封装,契约 api/openapi/user/billing.yaml + misc.yaml(coupons)。
 * 端点前缀 /api/user/v1,由 Api.base 提供。
 */
object BillApi {

    /** 支付/充值成功后暂存支付单号,PayResult 页无路由参数,经此传递。 */
    var lastPayNo: String? = null

    fun methodLabel(method: String): String = when (method) {
        "wechat" -> "微信支付"
        "alipay" -> "支付宝"
        "card" -> "银行卡"
        "cash" -> "现金"
        "prepaid" -> "预付券"
        else -> method
    }

    /** GET /bills?status=recent|paid|unpaid,返回 currentDue/currentPeriod/items。 */
    suspend fun bills(status: String? = null): JSONObject =
        Api.get("/bills" + Api.qs(mapOf("status" to status)))

    /** GET /bills/{billNo},返回 bill/items/totalDue/autoPayEnabled。 */
    suspend fun billDetail(billNo: String): JSONObject = Api.get("/bills/$billNo")

    /** POST /payments {billNo,amount,payMethod},返回 payNo/amount/billPeriod/payMethod/status。 */
    suspend fun createPayment(billNo: String, amount: Double, payMethod: String): JSONObject =
        Api.post("/payments", JSONObject().put("billNo", billNo).put("amount", amount).put("payMethod", payMethod))

    /** GET /payments,返回 items(PaymentRecord)。 */
    suspend fun payments(): JSONArray = Api.getArray("/payments")

    /** GET /payments/{payNo}/receipt,返回 receiptNo/customerName/amount/period/payMethod/paidAt。 */
    suspend fun receipt(payNo: String): JSONObject = Api.get("/payments/$payNo/receipt")

    /** GET /topups,返回 balance/denominations。 */
    suspend fun balance(): JSONObject = Api.get("/topups")

    /** POST /topups {amount,payMethod},返回 PaymentResult。 */
    suspend fun topup(amount: Double, payMethod: String): JSONObject =
        Api.post("/topups", JSONObject().put("amount", amount).put("payMethod", payMethod))

    /** GET /invoices,返回 titleType/title/taxNo/availablePeriods/records 聚合。 */
    suspend fun invoices(): JSONObject = Api.get("/invoices")

    /** POST /invoices {billNo},申请开票。 */
    suspend fun applyInvoice(billNo: String): JSONObject =
        Api.post("/invoices", JSONObject().put("billNo", billNo))

    /** GET /coupons?status=available|used|expired,返回 items/inviteLink。 */
    suspend fun coupons(status: String? = null): JSONObject =
        Api.get("/coupons" + Api.qs(mapOf("status" to status)))
}
