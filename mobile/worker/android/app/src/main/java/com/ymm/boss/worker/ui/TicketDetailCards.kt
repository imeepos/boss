package com.ymm.boss.worker.ui

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import org.json.JSONArray
import org.json.JSONObject

/**
 * 工单详情卡片集(对齐 designs/worker-order-detail-v1.spec.md):
 * - 工单头(按状态/类型分支)
 * - 快捷动作(3 列等分)
 * - 装维进度时间轴(横版,12/6 节点)
 * - 四码校验 2×2
 * - 风控校验
 * - 完成回执(D 屏专属)
 * - 底栏动作区(sticky)
 *
 * 类型(INSTALL/REPAIR)由 stages.length 推断(后端 TicketDetail 暂未返 type)。
 */

// Card1 工单头(状态/类型分支)
@Composable
internal fun DetailHeaderCard(d: JSONObject, type: String) {
    val status = d.optString("status")
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically) {
            Text(d.optString("ticketNo"), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(d.optString("statusLabel"), status)
        }
        if (type == "REPAIR") {
            val fault = d.optString("faultTypeLabel")
            if (fault.isNotEmpty()) Text(fault, fontSize = 15.sp, fontWeight = FontWeight.Bold,
                color = Color(0xFFFF4D4F), modifier = Modifier.padding(bottom = 4.dp))
        } else {
            val product = d.optString("product")
            if (product.isNotEmpty()) KvRow("产品", product)
        }
        val name = d.optString("customerName")
        val phone = d.optString("customerPhoneMasked")
        if (name.isNotEmpty() || phone.isNotEmpty()) {
            KvRow("客户", listOf(name, phone).filter { it.isNotEmpty() }.joinToString(" "))
        }
        val addr = d.optString("address")
        if (addr.isNotEmpty()) KvRow("安装地址", addr)
        if (type != "REPAIR") {
            val sp = d.optString("splitterPort")
            if (sp.isNotEmpty()) KvRow("分光器/端口", sp)
            val pb = d.optString("preBindTag")
            if (pb.isNotEmpty()) KvRow("预绑定标签", pb)
            val ss = d.optString("scheduleSlot")
            if (ss.isNotEmpty()) KvRow("预约时间", ss)
            val dist = d.optDouble("distanceKm", -1.0)
            if (status == "TODO" && dist >= 0) KvRow("距离", "%.1f km".format(dist), valueColor = Primary)
        } else {
            val reported = d.optString("reportedAt")
            if (reported.isNotEmpty()) KvRow("报障时间", reported)
            val sla = d.optInt("slaLeftMinutes", -1)
            if (sla >= 0) {
                KvRow("SLA 剩余", "%d:%02d".format(sla / 60, sla % 60),
                    valueColor = Color(0xFFFA8C16))
            }
            val diag = d.optString("remoteDiagnosis")
            if (diag.isNotEmpty()) {
                Text("远程诊断:$diag", fontSize = 12.sp, color = Muted,
                    modifier = Modifier.padding(top = 4.dp))
            }
        }
        if (status == "DONE") {
            val finished = d.optString("finishedAt")
            if (finished.isNotEmpty()) KvRow("完成时间", finished)
        }
    }
}

// Card2 快捷动作(3 列等分;TODO 态隐藏,只需"领取工单"主按钮)
@Composable
internal fun QuickActionRow(d: JSONObject, nav: NavHost, no: String) {
    val ctx = LocalContext.current
    val phone = d.optString("customerPhoneMasked")
    Card(Modifier.padding(12.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            ActionBtn("联系客户", Modifier.weight(1f)) {
                if (phone.isBlank()) toast(ctx, "号码已脱敏，请通过平台联系")
                else ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
            }
            ActionBtn("一键导航", Modifier.weight(1f)) { nav.push(Screen.Navi(no)) }
            ActionBtn("到点签到", Modifier.weight(1f)) { nav.push(Screen.Checkin(no)) }
        }
    }
}

// Card3 装维进度时间轴(横版一字排开)
@Composable
internal fun TimelineCard(d: JSONObject) {
    val stages = d.optJSONArray("stages") ?: JSONArray()
    val total = stages.length()
    val doneCount = (0 until total).count { stages.optJSONObject(it).optString("result") == "DONE" }
    Card(Modifier.padding(12.dp)) {
        SectionTitle("装维进度", more = "$doneCount/$total 已完成")
        Row(modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically) {
            for (i in 0 until total) {
                val st = nodeStateOf(stages.optJSONObject(i).optString("result"))
                NodeDot(st, stages.optJSONObject(i).optInt("stage"))
                if (i < total - 1) {
                    val next = nodeStateOf(stages.optJSONObject(i + 1).optString("result"))
                    TimelineConnector(st == NodeState.DONE && next == NodeState.DONE, Modifier.weight(1f))
                }
            }
        }
    }
}

// Card4 四码校验(2×2)
@Composable
internal fun QuadCard(quad: JSONObject?) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("四码校验", more = "一致性校验")
        if (quad == null) { Notice("四码信息缺失"); return@Card }
        val matched = quad.optBoolean("matched", true)
        val status = quad.optString("status")
        val stText = when {
            status == "CONFLICT" -> "不一致 · 需核实"
            matched -> "已通过校验"
            else -> "待校验"
        }
        val okColor = if (matched) Color(0xFF0AA847) else Color(0xFFFF4D4F)
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                QuadCell("资产码", quad.optString("assetCode"), stText, okColor, Modifier.weight(1f))
                QuadCell("用户码", quad.optString("customerCode"), stText, okColor, Modifier.weight(1f))
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                QuadCell("端口码", quad.optString("portCode"), stText, okColor, Modifier.weight(1f))
                QuadCell("地址码", quad.optString("addrCode"), stText, okColor, Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun QuadCell(label: String, value: String, st: String, color: Color, modifier: Modifier) {
    val border = when (color) {
        Color(0xFF0AA847) -> Color(0xFF6FD18B)
        Color(0xFFFF4D4F) -> Color(0xFFFF9B9D)
        else -> Color(0xFFE7EAF0)
    }
    Box(modifier = modifier.background(Color.White, RoundedCornerShape(8.dp))
        .border(1.dp, border, RoundedCornerShape(8.dp)).padding(12.dp)) {
        Column {
            Text(label, fontSize = 12.sp, color = Muted)
            Text(value.ifEmpty { "-" }, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Ink)
            Text(st, fontSize = 12.sp, color = color)
        }
    }
}

// Card5 风控校验
@Composable
internal fun RiskCard(risk: JSONObject?) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("风控校验", more = "名单拦截")
        val hit = risk?.optBoolean("blacklistHit", false) == true
                || risk?.optBoolean("graylistHit", false) == true
        val color = if (hit) Color(0xFFFF2D2F) else Color(0xFF0AA847)
        val text = if (hit) "已命中，自动拦截或转人工审核" else "未命中，可正常装维"
        Row(Modifier.fillMaxWidth().padding(vertical = 6.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text(if (hit) "⚠ 命中黑/灰名单" else "✓ 未命中名单", fontSize = 13.sp, color = color,
                fontWeight = FontWeight.Medium)
            Text(text, fontSize = 13.sp, color = color)
        }
        Notice("若命中风控名单，此单将自动拦截或转人工审核，操作全程留痕。")
    }
}

// Card6 完成回执(D 屏专属;签名位按 spec §6 不放预置签名)
@Composable
internal fun ReceiptCard() {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("完成回执")
        Row(verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.padding(vertical = 4.dp)) {
            Box(modifier = Modifier.background(Color(0xFF0AA847), androidx.compose.foundation.shape.CircleShape)
                .padding(4.dp)) {
                Text("✓", color = Color.White, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
            Text("客户已确认 · 服务完成", fontSize = 14.sp, fontWeight = FontWeight.SemiBold,
                color = Color(0xFF0AA847))
        }
        Spacer(Modifier.height(8.dp))
        Text("客户签名", fontSize = 12.sp, color = Muted)
        Box(modifier = Modifier.fillMaxWidth().height(56.dp)
            .border(1.dp, Color(0xFFD9D9D9), RoundedCornerShape(8.dp))
            .padding(8.dp), contentAlignment = Alignment.Center) {
            Text("（虚线框 · 占位）", fontSize = 12.sp, color = Muted)
        }
    }
}

// 底栏动作区:按状态分支(sticky)
@Composable
internal fun BottomActionBar(d: JSONObject, nav: NavHost, no: String,
                              onAccept: () -> Unit, onRollback: () -> Unit, onRetry: () -> Unit) {
    val status = d.optString("status")
    val type = inferTicketType(d)
    Row(modifier = Modifier.fillMaxWidth().background(Color.White)
        .padding(horizontal = 12.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        when {
            status == "TODO" -> PrimaryAction("领取工单", Modifier.weight(1f), onClick = onAccept)
            status == "DONE" -> {
                OutlinedAction("查看报告", Modifier.weight(1f)) { nav.push(Screen.Report(no)) }
                OutlinedAction("返回列表", Modifier.weight(1f)) { nav.pop() }
            }
            type == "REPAIR" -> {
                OutlinedAction("联系客户", Modifier.weight(1f)) { nav.push(Screen.Service) }
                OutlinedAction("改约", Modifier.weight(1f)) { nav.push(Screen.Reschedule(no)) }
                PrimaryAction("修复上报", Modifier.weight(1f)) { nav.push(Screen.RepairReport(no)) }
            }
            else -> {
                OutlinedAction("异常上报", Modifier.weight(1f)) { nav.push(Screen.ScanAbnormal(no)) }
                OutlinedAction("转单", Modifier.weight(1f)) { nav.push(Screen.Transfer(no)) }
                OutlinedAction("改约", Modifier.weight(1f)) { nav.push(Screen.Reschedule(no)) }
                PrimaryAction("扫码绑定", Modifier.weight(1f)) { nav.push(Screen.Scan(no)) }
            }
        }
    }
}

// A 屏时间轴卡内次按钮(回退/重试,spec §4.7)
@Composable
internal fun TimelineExtras(no: String, onRollback: () -> Unit, onRetry: () -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp)) {
        OutlinedAction("回退上一环节", Modifier.weight(1f), onClick = onRollback)
        OutlinedAction("重试失败环节", Modifier.weight(1f), onClick = onRetry)
    }
}