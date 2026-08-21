package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 工单列表(样式对齐 user 端账单页:胶囊筛选 tab + 白卡列表 + 居中空态)
private val SEGS = listOf("doing" to "进行中", "todo" to "待领取", "done" to "已完成", "all" to "全部")
private val TITLES = mapOf("doing" to "进行中工单", "todo" to "待领取工单", "done" to "已完成 · 今日", "all" to "全部工单")

@Composable
fun OrdersScreen(nav: NavHost) {
    var cur by remember { mutableStateOf("doing") }
    var refresh by remember { mutableStateOf(0) }
    var tip by remember { mutableStateOf("") }
    val state by loadOnce(cur, refresh) { TicketApi.list(if (cur == "all") null else cur.uppercase()) }
    val scope = rememberCoroutineScope()

    fun take(no: String) {
        scope.launch {
            tip = try { TicketApi.accept(no).optString("message", "已领取") }
            catch (e: Exception) { "领取失败，请重试。" }
            refresh++
        }
    }

    Column(Modifier.fillMaxSize()) {
        TopBar("工单列表", action = "刷新", onAction = { refresh++ })
        FilterTabs(cur) { cur = it }
        if (tip.isNotEmpty()) {
            Text(tip, fontSize = 12.sp, color = Color(0xFFCF1322), modifier = Modifier.padding(horizontal = 16.dp))
        }
        LazyColumn(Modifier.fillMaxSize()) {
            item {
                HomeCard(topPadding = 0) {
                    when (val s = state) {
                        is Load.Loading -> { CardTitle(TITLES[cur] ?: ""); Loading() }
                        is Load.Fail -> { CardTitle(TITLES[cur] ?: ""); Notice("工单加载失败，请刷新重试。") }
                        is Load.Ok -> TicketList(s.data.optJSONArray("items") ?: JSONArray(), nav) { take(it) }
                    }
                }
                Spacer(Modifier.height(16.dp))
            }
        }
    }
}

@Composable
private fun TicketList(items: JSONArray, nav: NavHost, onTake: (String) -> Unit) {
    CardTitle("工单 (${items.length()} 单)")
    if (items.length() == 0) {
        EmptyState("暂无工单")
        return
    }
    for (i in 0 until items.length()) {
        val t = items.optJSONObject(i) ?: continue
        TicketRow(t, onTake = onTake) { nav.push(ticketScreen(t.optString("ticketNo"))) }
    }
}

private fun sectionTitle(items: JSONArray): String = "工单 (${items.length()} 单)"

/** 胶囊筛选 tab(对齐 user 端 PillTab plain 形态:选中实心主色,未选中无底色)。 */
@Composable
private fun PillTab(label: String, active: Boolean, onClick: () -> Unit) {
    Text(
        label, fontSize = 13.sp, textAlign = TextAlign.Center,
        color = if (active) Color.White else Muted,
        fontWeight = if (active) FontWeight.W500 else FontWeight.Normal,
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(if (active) Primary else Color(0xFFEEF0F3))
            .border(1.dp, if (active) Primary else Color.Transparent, RoundedCornerShape(999.dp))
            .clickable { onClick() }
            .padding(horizontal = 14.dp, vertical = 7.dp),
    )
}

/** 列表行(对齐 user 端 BillCell:主标题 + 副标题 + 右侧状态列)。 */
@Composable
private fun TicketRow(t: JSONObject, onTake: (String) -> Unit, onClick: () -> Unit) {
    val no = t.optString("ticketNo")
    val dist = if (t.isNull("distanceKm")) "" else " · 距您 ${t.optDouble("distanceKm")}km"
    Row(
        Modifier.fillMaxWidth().clickable(enabled = t.optString("status") != "TODO") { onClick() }
            .padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text("$no · ${t.optString("typeLabel")}", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Ink)
            Text(t.optString("address") + dist, fontSize = 12.sp, color = Muted)
        }
        Column(horizontalAlignment = Alignment.End) {
            StatusTag(t.optString("statusLabel"), t.optString("status"))
            if (!t.isNull("stageTotal")) {
                Text("${t.optInt("stage", -1)}/${t.optInt("stageTotal")} 环节", fontSize = 12.sp, color = Muted)
            }
            if (t.optString("status") == "TODO") AcceptBtn { onTake(no) }
        }
    }
}

@Composable
private fun AcceptBtn(onTake: () -> Unit) {
    Text(
        "领取", fontSize = 13.sp, color = Color.White, textAlign = TextAlign.Center,
        modifier = Modifier
            .padding(top = 6.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(Primary)
            .clickable { onTake() }
            .padding(horizontal = 16.dp, vertical = 6.dp),
    )
}

/** 胶囊筛选 tab(对齐 user 端 PillTab plain 形态)。 */
@Composable
private fun FilterTabs(current: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        SEGS.forEach { (key, label) ->
            PillTab(label, active = key == current) { onSelect(key) }
        }
    }
}

/** 分段选择器(HistoryScreen 复用,保留原公共签名)。 */
@Composable
fun SegRow(cur: String, onSelect: (String) -> Unit, segs: List<Pair<String, String>> = SEGS) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp)
        .clip(RoundedCornerShape(9.dp))
        .background(Color(0xFFEEF0F3))
        .padding(3.dp)) {
        segs.forEach { (k, label) ->
            val active = k == cur
            Text(label, fontSize = 13.sp,
                color = if (active) Ink else Muted,
                fontWeight = if (active) FontWeight.SemiBold else FontWeight.Normal,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(7.dp))
                    .background(if (active) Color.White else Color.Transparent)
                    .clickable { onSelect(k) }
                    .padding(vertical = 7.dp))
        }
    }
}

/** 空状态:图标 + 文字居中(对齐 user 端 EmptyState)。 */
@Composable
private fun EmptyState(text: String) {
    Column(
        Modifier.fillMaxWidth().padding(vertical = 24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Outlined.Inbox, contentDescription = null, tint = Muted, modifier = Modifier.size(40.dp))
        Spacer(Modifier.height(10.dp))
        Text(text, fontSize = 13.sp, color = Muted)
    }
}
