package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.FaultApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 对应草稿 docs/user/faultdetail.html:报修进度(no) + 报障 6 环节时间轴。
@Composable
fun FaultDetailScreen(nav: Nav, no: String) {
    var detail by remember { mutableStateOf<JSONObject?>(null) }
    var loadErr by remember { mutableStateOf("") }

    LaunchedEffect(no) {
        try {
            detail = FaultApi.detail(no)
        } catch (e: Exception) { loadErr = "报修单加载失败" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("报修详情", { nav.pop() }, action = "评价") { nav.push(Route.Rate(no)) }
        val d = detail
        if (d == null) {
            AppCard { CardTitle(if (loadErr.isBlank()) "加载中…" else loadErr) }
        } else {
            val fault = d.optJSONObject("fault") ?: JSONObject()
            InfoCard(fault, d)
            TimelineCard(d.optJSONArray("timeline").toObjList())
            ActionsCard()
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun InfoCard(fault: JSONObject, d: JSONObject) {
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            CardTitle(fault.optString("ticketNo", "—"))
            Spacer(Modifier.width(8.dp))
            Tag(fault.optString("statusLabel", "—"), Palette.primary)
        }
        CellRow("故障类型", right = { InfoVal(fault.optString("faultTypeLabel", "—")) })
        CellRow("故障地址", right = { InfoVal(fault.optString("address", "—")) })
        CellRow("报修时间", right = { InfoVal(fault.optString("createdAt", "—")) })
        val tech = buildString {
            append(d.optString("technicianName").ifBlank { "—" })
            d.optString("technicianPhoneMasked").takeIf { it.isNotBlank() }?.let { append(" · ").append(it) }
        }
        CellRow("受理师傅", right = { InfoVal(tech) })
        CellRow("SLA", right = { InfoVal(d.optString("sla").ifBlank { "—" }) })
    }
}

@Composable
private fun InfoVal(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.ink, modifier = Modifier.padding(start = 12.dp))
}

@Composable
private fun TimelineCard(timeline: List<JSONObject>) {
    AppCard {
        CardTitle("处理进度（报障 6 环节）")
        if (timeline.isEmpty()) {
            Text("暂无进度", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
        }
        timeline.forEachIndexed { i, t ->
            val heading = t.optInt("step").let { if (it > 0) "$it " else "" } + t.optString("title")
            TimelineItem(heading, t.optString("result", "PENDING"), t.optString("meta"), isLast = i == timeline.size - 1)
        }
    }
}

@Composable
private fun TimelineItem(title: String, result: String, meta: String, isLast: Boolean) {
    Row(Modifier.fillMaxWidth().padding(top = 10.dp)) {
        Column(Modifier.width(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            val color = when (result) {
                "DONE" -> Palette.success
                "DOING" -> Palette.primary
                else -> Palette.subtle
            }
            androidx.compose.foundation.layout.Box(
                Modifier.size(10.dp).background(color, CircleShape),
            )
            if (!isLast) {
                Column(
                    Modifier.width(2.dp).height(30.dp)
                        .background(if (result == "DONE") Palette.success else Palette.line),
                ) {}
            }
        }
        Column(Modifier.padding(start = 10.dp)) {
            Text(title, fontSize = 13.5.sp, fontWeight = FontWeight.W500, color = Palette.ink)
            if (meta.isNotBlank()) Text(meta, fontSize = 12.sp, color = Palette.muted)
        }
    }
}

// TODO 契约仅有 GET /faults/{ticketNo},联系师傅与催单端点未定义,先按草稿摆按钮
@Composable
private fun ActionsCard() {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 6.dp)) {
        OutlinedButton(
            onClick = {
                // TODO 联系师傅:待端点
            },
            modifier = Modifier.weight(1f).height(42.dp),
        ) { Text("联系师傅", color = Palette.primary) }
        Spacer(Modifier.width(10.dp))
        Button(
            onClick = {
                // TODO 催单:待端点
            },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.weight(1f).height(42.dp),
        ) { Text("催单", color = Color.White) }
    }
}
