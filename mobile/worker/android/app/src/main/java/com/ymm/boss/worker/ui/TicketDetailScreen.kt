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

// 工单详情(对齐 docs/worker/order.html)
@Composable
fun TicketDetailScreen(nav: NavHost, no: String) {
    val state by loadOnce(no) { TicketApi.detail(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("工单详情", onBack = { nav.pop() }, action = "联系调度",
            onAction = { nav.push(Screen.Service) })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                DetailCard(d)
                LinkRow("",
                    onNavi = { nav.push(Screen.Navi(no)) },
                    onCheckin = { nav.push(Screen.Checkin(no)) })
                TimelineCard(d, onRollback = {
                    scope.launch {
                        try { toast(ctx, TicketApi.rollback(no).optString("message", "已回退上一环节")) }
                        catch (e: Exception) { toast(ctx, "操作失败，请重试。") }
                    }
                }, onRetry = {
                    scope.launch {
                        try { toast(ctx, TicketApi.retry(no).optString("message", "已重试失败环节")) }
                        catch (e: Exception) { toast(ctx, "操作失败，请重试。") }
                    }
                })
                QuadCard(d.optJSONObject("quad"))
                RiskCard(d.optJSONObject("riskCheck"))
                FlowActions(nav, no)
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun DetailCard(d: JSONObject) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text(d.optString("ticketNo"), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(d.optString("statusLabel"), d.optString("status"))
        }
        KvRow("产品", d.optString("product"))
        KvRow("客户", "${d.optString("customerName")}(${d.optString("customerPhoneMasked")})")
        KvRow("安装地址", d.optString("address"))
        KvRow("分光器/端口", d.optString("splitterPort"))
        KvRow("预绑定标签", d.optString("preBindTag"))
        KvRow("预约时间", d.optString("scheduleSlot"))
    }
}

@Composable
private fun LinkRow(phone: String, onNavi: () -> Unit, onCheckin: () -> Unit) {
    val ctx = LocalContext.current
    Card(Modifier.padding(12.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            ActionBtn("联系客户", Modifier.weight(1f)) {
                if (phone.isBlank()) toast(ctx, "号码已脱敏，请通过平台联系")
                else ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
            }
            ActionBtn("一键导航", Modifier.weight(1f), onClick = onNavi)
            ActionBtn("到点签到", Modifier.weight(1f), onClick = onCheckin)
        }
    }
}

@Composable
private fun TimelineCard(d: JSONObject, onRollback: () -> Unit, onRetry: () -> Unit) {
    Card(Modifier.padding(12.dp)) {
        val stages = d.optJSONArray("stages") ?: JSONArray()
        val done = (0 until stages.length()).count { stages.optJSONObject(it).optString("result") == "DONE" }
        SectionTitle("装维进度", more = "$done/${stages.length()} 已完成")
        for (i in 0 until stages.length()) TimelineItem(stages.optJSONObject(i))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.padding(top = 12.dp)) {
            ActionBtn("回退上一环节", Modifier.weight(1f), onClick = onRollback)
            ActionBtn("重试失败环节", Modifier.weight(1f), onClick = onRetry)
        }
    }
}

@Composable
private fun TimelineItem(stage: JSONObject) {
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
            Text("${stage.optInt("stage")} ${stage.optString("name")}",
                fontSize = 14.sp, fontWeight = FontWeight.Medium, color = Ink)
            val meta = listOfNotNull(
                stage.optString("finishedAt").takeIf { it.isNotEmpty() },
                stage.optString("note").takeIf { it.isNotEmpty() },
            ).joinToString(" · ")
            if (meta.isNotEmpty()) Text(meta, fontSize = 12.sp, color = Muted)
        }
    }
}

@Composable
private fun QuadCard(q: JSONObject?) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("预绑定四码对照", more = "一致性校验")
        if (q == null) { Notice("四码信息缺失", red = false); return@Card }
        val ok = q.optBoolean("matched", true)
        val st = if (ok) "已通过校验" else "不一致 · 需核实"
        QuadCell("资产码", q.optString("assetCode"), st, ok)
        QuadCell("用户码", q.optString("customerCode"), st, ok)
        QuadCell("端口码", q.optString("portCode"), st, ok)
        QuadCell("地址码", q.optString("addrCode"), st, ok)
    }
}

@Composable
private fun RiskCard(rk: JSONObject?) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("风控校验", more = "名单拦截")
        val hit = rk?.optBoolean("blacklistHit", false) == true || rk?.optBoolean("graylistHit", false) == true
        val text = if (hit) "已命中，自动拦截或转人工审核" else "未命中，可正常装维"
        val color = if (hit) Color(0xFFFF4D4F) else Success
        Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), horizontalArrangement = Arrangement.SpaceBetween) {
            Text("黑/灰名单", fontSize = 13.sp, color = Muted)
            Text(text, fontSize = 13.sp, color = color, fontWeight = FontWeight.Medium)
        }
        Notice("若命中风控名单，此单将自动拦截或转人工审核，操作全程留痕。")
    }
}

@Composable
private fun FlowActions(nav: NavHost, no: String) {
    Card(Modifier.padding(12.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            ActionBtn("异常上报", Modifier.weight(1f)) { nav.push(Screen.ScanAbnormal(no)) }
            ActionBtn("转单/改派", Modifier.weight(1f)) { nav.push(Screen.Transfer(no)) }
            ActionBtn("改约", Modifier.weight(1f)) { nav.push(Screen.Reschedule(no)) }
            ActionBtn("扫码绑定", Modifier.weight(1f), primary = true) { nav.push(Screen.Scan(no)) }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.padding(top = 10.dp)) {
            ActionBtn("激活重试", Modifier.weight(1f)) { nav.push(Screen.Activate(no)) }
            ActionBtn("拆机作业", Modifier.weight(1f)) { nav.push(Screen.Dismantle(no)) }
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

@Composable
private fun QuadCell(label: String, value: String, st: String, ok: Boolean) {
    val bg = if (ok) Color(0xFFF6FFED) else Color(0xFFFFF1F0)
    val fg = if (ok) Color(0xFF52C41A) else Color(0xFFFF4D4F)
    Box(Modifier.fillMaxWidth().padding(vertical = 4.dp)
        .background(bg, RoundedCornerShape(10.dp)).padding(12.dp)) {
        Column {
            Text(label, fontSize = 12.sp, color = Muted)
            Text(value, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Ink)
            Text(st, fontSize = 12.sp, color = fg)
        }
    }
}