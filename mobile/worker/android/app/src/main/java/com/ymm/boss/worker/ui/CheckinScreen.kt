package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 到点签到(对齐 docs/worker/checkin.html)
@Composable
fun CheckinScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("到点签到", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("客户", "${d.optString("customerName")}(${d.optString("customerPhoneMasked")})")
                    KvRow("地址", d.optString("address"))
                    KvRow("预约时段", d.optString("scheduleSlot"))
                }
                Card(Modifier.padding(12.dp)) {
                    Notice("到达现场后签到，系统记录定位与时间，超时未签到将触发调度预警。")
                }
                Card(Modifier.padding(12.dp)) {
                    PrimaryButton("确认签到", modifier = Modifier.fillMaxWidth()) {
                        scope.launch {
                            try {
                                val r = TicketApi.checkin(no, 0.0, 0.0)
                                toast(ctx, r.optString("message", "签到成功"))
                                nav.pop()
                            } catch (e: Exception) { toast(ctx, "签到失败：${e.message}") }
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}