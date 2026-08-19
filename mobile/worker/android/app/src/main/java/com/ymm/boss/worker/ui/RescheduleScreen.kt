package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 改约(对齐 docs/worker/reschedule.html)
@Composable
fun RescheduleScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val slots = listOf("09:00-11:00", "14:00-16:00", "16:00-18:00")
    val reasons = listOf("客户要求改期", "客户不在家（爽约）", "师傅排期冲突", "天气/交通延误")
    var date by remember { mutableStateOf("") }
    var slot by remember { mutableStateOf(slots.first()) }
    var reason by remember { mutableStateOf(reasons.first()) }
    var remark by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("改约", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("客户", "${d.optString("customerName")} · ${d.optString("customerPhoneMasked")}")
                    KvRow("原预约", d.optString("scheduleSlot", "--"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("新预约日期")
            OutlinedTextField(value = date, onValueChange = { date = it },
                placeholder = { Text("如 2026-08-20") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel("新预约时段")
            OptionRow(slots, slot) { slot = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel("变更原因")
            OptionRow(reasons, reason) { reason = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel("备注（选填）")
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text("特殊情况说明") }, minLines = 2,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton("提交改约", modifier = Modifier.fillMaxWidth()) {
                if (date.isBlank()) { tip = "请填写新预约日期"; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = TicketApi.reschedule(no, date.trim(), slot, reason, remark.trim())
                        toast(ctx, r.optString("message", "改约成功"))
                        nav.pop()
                        ""
                    } catch (e: Exception) { "改约失败：${e.message}" }
                }
            }
            if (tip.isNotEmpty()) Text(tip, color = androidx.compose.ui.graphics.Color(0xFFCF1322),
                fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
        }
        Card(Modifier.padding(12.dp)) {
            Notice("改约与「爽约」同一入口：客户不在家选「客户不在家」，系统二次短信确认并重新计时 SLA。")
        }
        Spacer(Modifier.height(12.dp))
    }
}