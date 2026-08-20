package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
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

// 工单列表(对齐 docs/worker/orders.html):进行中/待领取/已完成/全部 分段
private val SEGS = listOf("doing" to "进行中", "todo" to "待领取", "done" to "已完成", "all" to "全部")
private val TITLES = mapOf("doing" to "进行中工单", "todo" to "待领取工单", "done" to "已完成 · 今日", "all" to "全部工单")

@Composable
fun OrdersScreen(nav: NavHost) {
    var cur by remember { mutableStateOf("doing") }
    var refresh by remember { mutableStateOf(0) }
    var tip by remember { mutableStateOf("") }
    val state by loadOnce(cur, refresh) {
        TicketApi.list(if (cur == "all") null else cur.uppercase())
    }
    val scope = rememberCoroutineScope()

    fun take(no: String) {
        scope.launch {
            tip = try {
                val r = TicketApi.accept(no)
                r.optString("message", "已领取")
            } catch (e: Exception) { "领取失败，请重试。" }
            refresh++
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("工单列表", action = "刷新", onAction = { refresh++ })
        SegRow(cur, onSelect = { cur = it })
        if (tip.isNotEmpty()) { Text(tip, fontSize = 12.sp, color = Color(0xFFCF1322), modifier = Modifier.padding(horizontal = 14.dp)) }
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> { SectionTitle("${TITLES[cur]} · 加载中…"); Loading() }
                is Load.Fail -> { SectionTitle(TITLES[cur] ?: ""); Notice("工单加载失败，请刷新重试。") }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    SectionTitle("${TITLES[cur]} (${items.length()} 单)")
                    if (items.length() == 0) Empty("暂无工单")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        if (it0.optString("status") == "TODO") {
                            TicketCell(it0, rightExtra = { AcceptBtn { take(it0.optString("ticketNo")) } })
                        } else {
                            TicketCell(it0, onClick = { nav.push(ticketScreen(it0.optString("ticketNo"))) })
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AcceptBtn(onTake: () -> Unit) {
    Text("领取", fontSize = 13.sp, color = Color.White,
        modifier = Modifier
            .padding(top = 4.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(Primary)
            .clickable { onTake() }
            .padding(horizontal = 12.dp, vertical = 6.dp))
}

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
