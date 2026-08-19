package com.ymm.boss.worker.ui

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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.HomeApi
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import org.json.JSONObject
import java.time.LocalDate

// 快捷入口(对齐 home.html 宫格:排期/公告/任务池/手册/测速/领料/安全/维护)
private data class QuickItem(val title: String, val sub: String, val screen: Screen)

private val QUICK = listOf(
    QuickItem("排期", "日程", Screen.Schedule), QuickItem("公告", "通告", Screen.Notice),
    QuickItem("任务池", "抢单", Screen.Hall), QuickItem("手册", "排障", Screen.Help),
    QuickItem("测速", "工具", Screen.Tool), QuickItem("领料", "出库", Screen.Pickup),
    QuickItem("安全", "上报", Screen.Safety), QuickItem("维护", "清单", Screen.Maintenance),
)

// 工作台(对齐 docs/worker/home.html)
@Composable
fun HomeScreen(nav: NavHost) {
    val state by loadOnce { HomeApi.get() }
    val msgs by loadOnce { MiscApi.messages() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        when (val s = state) {
            is Load.Loading -> { Head("加载中…", "", "今日接单 - · 进行中 - · 待处理 -"); Loading() }
            is Load.Fail -> {
                Head("工人", "", "")
                Card(Modifier.padding(14.dp)) { Notice("数据加载失败，请刷新重试。") }
            }
            is Load.Ok -> HomeBody(nav, s.data)
        }
        NoticeList(msgs)
        QuickGrid(nav)
    }
}

@Composable
private fun HomeBody(nav: NavHost, d: JSONObject) {
    val today = d.optJSONObject("today") ?: JSONObject()
    Head(d.optString("workerName", "工人"),
        "${d.optString("groupName")} · ${d.optString("phoneMasked")}",
        "今日接单 ${today.optInt("accepted")} · 进行中 ${today.optInt("doing")} · 待处理 ${today.optInt("todo")}")
    Box(Modifier.padding(horizontal = 14.dp)) {
        StatCard("今日业绩", LocalDate.now().toString(), listOf(
            Triple("${today.optInt("accepted")}", "接单", Primary),
            Triple("${today.optInt("finished")}", "已完成", Color(0xFF52C41A)),
            Triple("${today.optInt("doing")}", "进行中", Color(0xFFFAAD14)),
        ))
    }
    val ongoing = d.optJSONArray("ongoing") ?: org.json.JSONArray()
    Card(Modifier.padding(top = 24.dp)) {
        SectionTitle("进行中工单 (${ongoing.length()})", more = "全部") { nav.switchTab(Screen.Orders) }
        if (ongoing.length() == 0) Empty("暂无进行中工单")
        for (i in 0 until ongoing.length()) {
            TicketCell(ongoing.optJSONObject(i), onClick = { nav.push(ticketScreen(ongoing.optJSONObject(i).optString("ticketNo"))) })
        }
    }
}

@Composable
private fun Head(name: String, sub: String, statusLine: String) {
    Column(Modifier.fillMaxWidth()
        .background(Brush.linearGradient(listOf(Primary, Primary2)), RoundedCornerShape(bottomStart = 22.dp, bottomEnd = 22.dp))
        .padding(horizontal = 16.dp, vertical = 24.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(name, color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.Bold)
            Text("◉", color = Color.White, fontSize = 18.sp,
                modifier = Modifier.padding(4.dp))
        }
        if (sub.isNotEmpty()) Text(sub, color = Color.White.copy(alpha = .85f), fontSize = 13.sp)
        Spacer(Modifier.height(14.dp))
        StatusLine(statusLine)
    }
}

@Composable
private fun NoticeList(msgs: Load<JSONObject>) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("今日提醒", more = "消息中心")
        when (msgs) {
            is Load.Loading -> Text("加载中…", fontSize = 13.sp, color = Muted)
            is Load.Fail -> Notice("消息加载失败，请刷新重试。")
            is Load.Ok -> {
                val items = msgs.data.optJSONArray("items") ?: org.json.JSONArray()
                if (items.length() == 0) Empty("暂无提醒")
                for (i in 0 until minOf(items.length(), 3)) {
                    val x = items.optJSONObject(i)
                    KvRow(x.optString("title"), x.optString("content"))
                }
            }
        }
    }
}

@Composable
fun KvRow(k: String, v: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(k, fontSize = 13.sp, color = Muted)
        Text(v, fontSize = 13.sp, color = Ink, fontWeight = FontWeight.Medium)
    }
}

@Composable
private fun QuickGrid(nav: NavHost) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("快捷入口")
        QUICK.chunked(4).forEach { row ->
            Row(Modifier.fillMaxWidth().padding(vertical = 6.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                row.forEach { it0 ->
                    Column(Modifier.weight(1f)
                        .clip(RoundedCornerShape(10.dp))
                        .background(Color(0xFFFAFAFA), RoundedCornerShape(10.dp))
                        .clickable { nav.push(it0.screen) }
                        .padding(vertical = 12.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(it0.title, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Ink)
                        Text(it0.sub, fontSize = 11.sp, color = Muted)
                    }
                }
            }
        }
    }
    Spacer(Modifier.height(12.dp))
    Box(Modifier.fillMaxWidth().height(1.dp).background(Line))
}
