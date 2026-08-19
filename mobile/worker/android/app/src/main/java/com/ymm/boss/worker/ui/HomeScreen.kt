package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
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

// 快捷入口(对齐 home.html 宫格;目标页未开发前仅展示)
private val QUICK = listOf(
    "排期" to "日程", "公告" to "通告", "任务池" to "抢单", "手册" to "排障",
    "测速" to "工具", "领料" to "出库", "安全" to "上报", "维护" to "清单",
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
        QuickGrid()
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
            TicketCell(ongoing.optJSONObject(i)) { nav.push(Screen.TicketDetail(ongoing.optJSONObject(i).optString("ticketNo"))) }
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
private fun QuickGrid() {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("快捷入口")
        QUICK.chunked(4).forEach { row ->
            Row(Modifier.fillMaxWidth().padding(vertical = 6.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                row.forEach { (b, s) ->
                    Column(Modifier.weight(1f)
                        .background(Color(0xFFFAFAFA), RoundedCornerShape(10.dp))
                        .padding(vertical = 12.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(b, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Ink)
                        Text(s, fontSize = 11.sp, color = Muted)
                    }
                }
            }
        }
    }
    Spacer(Modifier.height(12.dp))
    Box(Modifier.fillMaxWidth().height(1.dp).background(Line))
}
