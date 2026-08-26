package com.ymm.boss.user.ui

/** 全部页面路由,与 docs/user 下的草稿 html 一一对应;详情页参数对应草稿 URL query。 */
sealed interface Route {
    data object Login : Route
    data object Register : Route
    data object Forgot : Route
    data object Verify : Route
    data object Agreement : Route
    data object Home : Route
    data object Products : Route
    data class Product(val id: String) : Route
    /** 订单确认页:套餐确认 + 安装地址选择 + Stripe 支付,productId 进栈后服务端建账并跳 PaymentSheet。 */
    data class OrderConfirm(val id: String) : Route
    data object Orders : Route
    data class Order(val no: String) : Route
    data class Rate(val no: String) : Route
    data class OrderChangeAddress(val no: String) : Route
    data object MyPlan : Route
    data class Change(val planId: String) : Route
    data class Cancel(val planId: String) : Route
    data class Move(val planId: String) : Route
    data object Bills : Route
    data class Bill(val no: String) : Route
    data object Pay : Route
    data object PayResult : Route
    data object Topup : Route
    data object Invoice : Route
    data object Coupon : Route
    data class Receipt(val payNo: String) : Route
    data object Fault : Route
    data class FaultDetail(val no: String) : Route
    data object Complaint : Route
    /** 投诉详情页:ticketNo 即 complaintId,展示描述 + cs_ticket_events 派生处理历史。 */
    data class ComplaintDetail(val ticketNo: String) : Route
    data object Service : Route
    data object Diy : Route
    data object Help : Route
    data object Messages : Route
    data object Notify : Route
    data object Profile : Route
    data object Security : Route
    data object Usage : Route
    data object Address : Route
    data object Addon : Route
}
