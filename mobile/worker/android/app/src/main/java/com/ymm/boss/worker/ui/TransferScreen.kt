package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 转单(对齐 docs/worker/transfer.html)
@Composable
fun TransferScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val reasons = listOf("现场条件不满足", "客户要求换人", "设备/材料需更换", "其他")
    var reason by remember { mutableStateOf(reasons.first()) }
    var target by remember { mutableStateOf("") }
    var remark by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("转单", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("预约时段", d.optString("scheduleSlot", "--"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("转单原因")
            OptionRow(reasons, reason) { reason = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel("转派对象（选填，留空退回调度池）")
            OutlinedTextField(value = target, onValueChange = { target = it },
                placeholder = { Text("如：李师傅 · 装机一组") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel("说明")
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text("向调度说明现场情况") }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton("确认转单", modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    tip = try {
                        val targetId = target.trim().takeIf { it.isNotEmpty() }?.toLongOrNull()
                        if (target.trim().isNotEmpty() && targetId == null) {
                            tip = "转派对象请输入师傅 ID"
                            return@launch
                        }
                        val r = TicketApi.transfer(no, reason, targetId, remark.trim())
                        toast(ctx, r.optString("message", "转单成功"))
                        nav.pop()
                        ""
                    } catch (e: Exception) { "转单失败：${e.message}" }
                }
            }
            if (tip.isNotEmpty()) Text(tip, color = androidx.compose.ui.graphics.Color(0xFFCF1322),
                fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
        }
        Card(Modifier.padding(12.dp)) {
            Notice("转单后本单将从「我的工单」移除，避免同一订单双人处理。")
        }
        Spacer(Modifier.height(12.dp))
    }
}

// 单选选项行(横排)
@Composable
fun OptionRow(options: List<String>, selected: String, onSelect: (String) -> Unit) {
    Column {
        options.forEach { opt ->
            Row(Modifier.fillMaxWidth().clickable { onSelect(opt) }.padding(vertical = 10.dp),
                horizontalArrangement = Arrangement.SpaceBetween) {
                Text(opt, fontSize = 14.sp, color = androidx.compose.ui.graphics.Color(0xFF1F2329))
                Text(if (opt == selected) "●" else "○", fontSize = 14.sp,
                    color = if (opt == selected) androidx.compose.ui.graphics.Color(0xFF1677FF)
                    else androidx.compose.ui.graphics.Color(0xFFB0B3B8))
            }
        }
    }
}