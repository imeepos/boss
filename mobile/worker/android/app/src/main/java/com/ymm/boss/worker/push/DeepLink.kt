package com.ymm.boss.worker.push

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

// 通知点击深链:通知拉起时可能尚未登录/导航栈未就绪,先记 pending,
// AppRoot 观察该状态,登录态就绪后一次性消费并跳转工单详情。
object DeepLink {
    const val EXTRA_TICKET_NO = "ticket_no"

    var pendingNo by mutableStateOf<String?>(null)
        private set

    fun request(no: String?) {
        if (!no.isNullOrBlank()) pendingNo = no
    }

    fun consume(): String? {
        val n = pendingNo
        pendingNo = null
        return n
    }
}
