package com.ymm.boss.worker.ui

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
import androidx.compose.foundation.background
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Ink
import org.json.JSONArray
import org.json.JSONObject

// 工单详情(骨架版,对齐 docs/worker/order.html 的信息卡 + 环节时间轴;
// 扫码/激活/收费等操作入口由后续页面轮次补齐)
@Composable
fun TicketDetailScreen(nav: NavHost, no: String) {
    val state by loadOnce(no) { TicketApi.detail(no) }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("工单详情", onBack = { nav.pop() })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("详情加载失败，请返回重试。", red = true) }
            is Load.Ok -> DetailBody(s.data)
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun DetailBody(d: JSONObject) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle("${d.optString("ticketNo")} · ${d.optString("typeLabel")}")
        KvRow("产品", d.optString("product"))
        KvRow("客户", "${d.optString("customerName")}(${d.optString("customerPhoneMasked")})")
        KvRow("地址", d.optString("address"))
        KvRow("预约时段", d.optString("scheduleSlot"))
        KvRow("SLA 剩余", "${d.optInt("slaLeftMinutes")} 分钟")
    }
    Card(Modifier.padding(12.dp)) {
        SectionTitle("服务环节(${d.optInt("stage")}/${d.optInt("stageTotal")})")
        val stages = d.optJSONArray("stages") ?: JSONArray()
        for (i in 0 until stages.length()) TimelineItem(stages.optJSONObject(i))
    }
}

@Composable
private fun TimelineItem(stage: JSONObject) {
    val result = stage.optString("result") // done/doing/todo
    val dot = when (result) {
        "done" -> Color(0xFF52C41A)
        "doing" -> Color(0xFF1677FF)
        "err" -> Color(0xFFFF4D4F)
        else -> Color(0xFFD9D9D9)
    }
    Row(Modifier.fillMaxWidth().padding(bottom = 18.dp), verticalAlignment = Alignment.Top) {
        Spacer(Modifier.size(12.dp).background(dot, CircleShape))
        Spacer(Modifier.size(12.dp).height(0.dp))
        Column(Modifier.padding(start = 8.dp)) {
            Text(stage.optString("name"), fontSize = 14.sp, fontWeight = FontWeight.Medium, color = Ink)
            val meta = listOfNotNull(
                stage.optString("finishedAt").takeIf { it.isNotEmpty() },
                stage.optString("note").takeIf { it.isNotEmpty() },
            ).joinToString(" · ")
            if (meta.isNotEmpty()) Text(meta, fontSize = 12.sp, color = Muted)
        }
    }
}
