package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
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
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.ComplaintApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/**
 * 投诉详情页:基本信息卡 + 处理历史时间轴(由 cs_ticket_events 派生)。
 * 第 1 项固定为提交投诉节点(DONE),其后每条对应一次状态变更/升级事件。
 * 工单不存在或越权时显示空态 + 重试入口。
 */
@Composable
fun ComplaintDetailScreen(nav: Nav, ticketNo: String) {
    var detail by remember { mutableStateOf<JSONObject?>(null) }
    var timeline by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var loadErr by remember { mutableStateOf("") }

    LaunchedEffect(ticketNo, nav.refreshTick) {
        try {
            val resp = ComplaintApi.detail(ticketNo)
            detail = resp.optJSONObject("complaint")
            timeline = resp.optJSONArray("timeline").toObjList()
            loadErr = ""
        } catch (e: Exception) {
            // 404(归属不匹配)按"不存在"显示,避免泄漏存在性
            loadErr = if (e is Api.HttpError && e.status == 404) "" else "详情加载失败," + Api.friendlyMessage(e)
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("投诉详情", onBack = { nav.pop() })
        if (loadErr.isNotBlank()) Notice(loadErr, Palette.err)

        val d = detail
        when {
            d == null && loadErr.isBlank() -> AppCard { CardTitle("加载中…") }
            d == null -> AppCard { EmptyState("工单不存在或已删除") }
            else -> {
                InfoCard(d)
                TimelineCard(timeline)
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun InfoCard(c: JSONObject) {
    val status = c.optString("status", "OPEN")
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            CardTitle(c.optString("complaintId", "—"))
            Spacer(Modifier.width(8.dp))
            Tag(c.optString("statusLabel", "受理中"), statusTagColor(status))
        }
        CellRow("类型", right = { InfoVal(c.optString("typeLabel", "—")) })
        CellRow("提交时间", right = { InfoVal(c.optString("createdAt", "—")) })
        if (c.optString("relOrderNo").isNotBlank()) {
            CellRow("关联订单", right = { InfoVal(c.optString("relOrderNo")) })
        }
        if (c.optString("contact").isNotBlank()) {
            CellRow("联系方式", right = { InfoVal(c.optString("contact")) })
        }
        if (c.optString("closedAt").isNotBlank()) {
            CellRow("关闭时间", right = { InfoVal(c.optString("closedAt")) })
        }
        Spacer(Modifier.height(8.dp))
        Text("描述", fontSize = 13.sp, color = Palette.muted, modifier = Modifier.padding(bottom = 4.dp))
        Text(
            c.optString("description").ifBlank { "—" },
            fontSize = 13.sp, lineHeight = 18.sp, color = Palette.ink,
        )
    }
}

@Composable
private fun InfoVal(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.ink, modifier = Modifier.padding(start = 12.dp))
}

@Composable
private fun TimelineCard(timeline: List<JSONObject>) {
    AppCard {
        CardTitle("处理历史")
        if (timeline.isEmpty()) {
            EmptyState("暂无处理记录")
            return@AppCard
        }
        timeline.forEachIndexed { i, t ->
            TimelineItem(
                step = t.optInt("step", i + 1),
                title = t.optString("title"),
                meta = t.optString("meta"),
                note = t.optString("note"),
                eventType = t.optString("eventType"),
                isLast = i == timeline.size - 1,
            )
        }
    }
}

@Composable
private fun TimelineItem(
    step: Int,
    title: String,
    meta: String,
    note: String,
    eventType: String,
    isLast: Boolean,
) {
    Row(Modifier.fillMaxWidth().padding(top = 12.dp)) {
        Column(Modifier.width(20.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                Modifier.size(10.dp).background(Palette.success, CircleShape),
            )
            if (!isLast) {
                Box(
                    Modifier.width(2.dp).height(36.dp).background(Palette.success.copy(alpha = 0.4f)),
                )
            }
        }
        Column(Modifier.padding(start = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    "$step. $title",
                    fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink,
                )
            }
            if (meta.isNotBlank()) {
                Text(meta, fontSize = 12.sp, color = Palette.muted)
            }
            if (note.isNotBlank() && eventType != "SUBMITTED") {
                Text(
                    note, fontSize = 12.sp, color = Palette.muted,
                    modifier = Modifier.padding(top = 2.dp),
                )
            }
        }
    }
}

private fun statusTagColor(status: String) = when (status) {
    "CLOSED" -> Palette.muted
    "PROCESSING" -> Palette.warn
    else -> Palette.primary
}