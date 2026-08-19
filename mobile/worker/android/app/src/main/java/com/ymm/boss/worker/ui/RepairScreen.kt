package com.ymm.boss.worker.ui

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 修复工单(对齐 docs/worker/repair.html):TKT/EMG 前缀工单详情
@Composable
fun RepairScreen(nav: NavHost, no: String) {
    val state by loadOnce(no) { TicketApi.detail(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("修复工单", onBack = { nav.pop() }, action = "联系调度",
            onAction = { nav.push(Screen.Service) })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                RepairInfoCard(d)
                DiagCard(d)
                RepairTimeline(d)
                RepairActions(nav, no, d.optString("customerPhoneMasked").replace("*", "")) {
                    scope.launch {
                        try {
                            val r = TicketApi.repairReport(no, "FIXED", "")
                            toast(ctx, if (r.optBoolean("reviewPassed")) "网络已恢复，工单关闭，系统自动复核通过" else "结果已上报，系统复核中")
                            nav.pop()
                        } catch (e: Exception) { toast(ctx, "上报失败，请重试。") }
                    }
                }
                Notice("上报修复后系统自动复核网络是否恢复；未恢复将回「处理中」，恢复则关闭并触发满意度回访。")
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun RepairInfoCard(d: JSONObject) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text(d.optString("ticketNo"), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(d.optString("statusLabel"), d.optString("status"))
        }
        KvRow("故障类型", d.optString("faultTypeLabel", "-"))
        KvRow("客户", "${d.optString("customerName", "-")} · ${d.optString("customerPhoneMasked", "-")}")
        KvRow("报障时间", d.optString("reportedAt", "-"))
        KvRow("SLA 剩余", "${d.optInt("slaLeftMinutes")} 分钟")
        KvRow("地址", d.optString("address"))
        KvRow("分光器/端口", d.optString("splitterPort", "-"))
    }
}

@Composable
private fun DiagCard(d: JSONObject) {
    val diag = d.optString("remoteDiagnosis", "").takeIf { it.isNotEmpty() } ?: return
    Card(Modifier.padding(12.dp)) {
        SectionTitle("远程诊断", more = "结果可查")
        Text(diag, fontSize = 13.sp, color = Color(0xFF389E0D))
    }
}

@Composable
private fun RepairTimeline(d: JSONObject) {
    Card(Modifier.padding(12.dp)) {
        val stages = d.optJSONArray("stages") ?: JSONArray()
        val done = (0 until stages.length()).count { stages.optJSONObject(it).optString("result") == "DONE" }
        SectionTitle("修复流程", more = "报障闭环 · $done/${stages.length()} 完成")
        for (i in 0 until stages.length()) RepairStage(stages.optJSONObject(i))
    }
}

@Composable
private fun RepairStage(stage: JSONObject) {
    val result = stage.optString("result")
    val dot = when (result) {
        "DONE" -> Success
        "DOING" -> Color(0xFF1677FF)
        "ERROR" -> Color(0xFFFF4D4F)
        else -> Color(0xFFD9D9D9)
    }
    Row(Modifier.fillMaxWidth().padding(bottom = 16.dp), verticalAlignment = Alignment.Top) {
        Box(Modifier.size(12.dp).background(dot, CircleShape))
        Column(Modifier.padding(start = 10.dp)) {
            Text(stage.optString("name"), fontSize = 14.sp, fontWeight = FontWeight.Medium, color = Ink)
            val meta = listOfNotNull(
                stage.optString("finishedAt").takeIf { it.isNotEmpty() },
                stage.optString("note").takeIf { it.isNotEmpty() },
            ).joinToString(" · ")
            if (meta.isNotEmpty()) Text(meta, fontSize = 12.sp, color = Muted)
        }
    }
}

@Composable
private fun RepairActions(nav: NavHost, no: String, phone: String, onReport: () -> Unit) {
    val ctx = LocalContext.current
    Card(Modifier.padding(12.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            ActionBtn("一键导航", Modifier.weight(1f)) { nav.push(Screen.Navi(no)) }
            ActionBtn("到点签到", Modifier.weight(1f)) { nav.push(Screen.Checkin(no)) }
            ActionBtn("联系客户", Modifier.weight(1f)) {
                ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
            }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.padding(top = 10.dp)) {
            ActionBtn("现场测速", Modifier.weight(1f)) { nav.push(Screen.Tool(no)) }
            ActionBtn("上报修复", Modifier.weight(1f), primary = true, onClick = onReport)
        }
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.padding(top = 10.dp)) {
            ActionBtn("转单/改派", Modifier.weight(1f)) { nav.push(Screen.Transfer(no)) }
            ActionBtn("改约", Modifier.weight(1f)) { nav.push(Screen.Reschedule(no)) }
            ActionBtn("投诉登记", Modifier.weight(1f)) { nav.push(Screen.Complaint(no)) }
        }
    }
}

@Composable
private fun ActionBtn(text: String, modifier: Modifier = Modifier, primary: Boolean = false, onClick: () -> Unit) {
    val bg = if (primary) Color(0xFF1677FF) else Color.White
    val fg = if (primary) Color.White else Ink
    Text(text, fontSize = 13.sp, color = fg,
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(bg)
            .clickable { onClick() }
            .padding(vertical = 10.dp),
        textAlign = TextAlign.Center)
}