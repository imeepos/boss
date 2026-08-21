package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.Warn
import org.json.JSONObject

// 工单卡片(布局对齐 user 端 OrderCard:单号行 + 图标瓦片主体 + 里程碑进度条/完成态)

internal fun ticketStatusColor(status: String) = when (status) {
    "TODO" -> Warn
    "ACCEPTED", "DOING" -> Primary
    "DONE" -> Success
    else -> Muted
}

/** 12 环节 → 4 里程碑:与 user 端 OrderCard.milestoneOf 同一映射。 */
internal fun milestoneOf(stage: Int) = when {
    stage <= 1 -> 1
    stage <= 7 -> 2
    stage <= 11 -> 3
    else -> 4
}

@Composable
internal fun TicketOrderCard(t: JSONObject, onTake: (String) -> Unit, onClick: () -> Unit) {
    val no = t.optString("ticketNo")
    val status = t.optString("status")
    // 卡片容器对齐 user 端 AppCard:外边距 14/6 + 平面白底 + 12dp 圆角 + 16dp 内边距,无阴影
    Column(
        Modifier.fillMaxWidth()
            .padding(horizontal = 14.dp, vertical = 6.dp)
            .background(Color.White, RoundedCornerShape(12.dp))
            .padding(16.dp),
    ) {
        Column(Modifier.fillMaxWidth().clickable(enabled = status != "TODO") { onClick() }) {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(no, fontSize = 14.sp, fontWeight = FontWeight.W600, color = Ink)
                Text("${t.optInt("stage", 0)}/${t.optInt("stageTotal", 12)} 环节", fontSize = 12.sp, color = Muted)
            }
            Spacer(Modifier.height(10.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                IconTile(Icons.Filled.Home, Primary, 48.dp)
                Spacer(Modifier.width(12.dp))
                Column(Modifier.weight(1f)) {
                    Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            titleOf(t), fontSize = 15.sp, fontWeight = FontWeight.W600,
                            color = Ink, maxLines = 1, modifier = Modifier.weight(1f, fill = false),
                        )
                        Text(t.optString("statusLabel"), fontSize = 13.sp, color = ticketStatusColor(status))
                    }
                    Spacer(Modifier.height(6.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Filled.LocationOn, contentDescription = null, tint = Muted, modifier = Modifier.size(14.dp))
                        Spacer(Modifier.width(4.dp))
                        Text(
                            t.optString("address").ifEmpty { "地址待补充" },
                            fontSize = 12.sp, color = Muted, maxLines = 1,
                        )
                    }
                }
            }
            when (status) {
                "DONE" -> DoneFooter()
                "TODO" -> TodoFooter(no, onTake)
                else -> ProgressBody(t.optInt("stage", 1))
            }
        }
    }
}

private fun titleOf(t: JSONObject): String {
    val customer = t.optString("customerName")
    val type = t.optString("typeLabel")
    return listOf(customer, type).filter { it.isNotEmpty() }.joinToString(" · ").ifEmpty { "工单作业" }
}

@Composable
private fun ProgressBody(stage: Int) {
    Spacer(Modifier.height(14.dp))
    TicketStepper(milestoneOf(stage))
}

@Composable
private fun TodoFooter(no: String, onTake: (String) -> Unit) {
    Spacer(Modifier.height(12.dp))
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.End,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            "领取工单", fontSize = 13.sp, color = Color.White, textAlign = TextAlign.Center,
            modifier = Modifier
                .background(Primary, RoundedCornerShape(8.dp))
                .clickable { onTake(no) }
                .padding(horizontal = 20.dp, vertical = 8.dp),
        )
    }
}

@Composable
private fun DoneFooter() {
    Spacer(Modifier.height(12.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(Icons.Filled.CheckCircle, contentDescription = null, tint = Success, modifier = Modifier.size(20.dp))
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            Text("工单已完成", fontSize = 14.sp, fontWeight = FontWeight.W600, color = Ink)
            Text("装维完成,感谢师傅辛苦!", fontSize = 12.sp, color = Muted)
        }
    }
}

@Composable
private fun TicketStepper(current: Int) {
    val labels = listOf("提交订单", "受理成功", "上门安装", "完成")
    Box(Modifier.fillMaxWidth().height(52.dp)) {
        Row(
            Modifier.fillMaxWidth().padding(start = 36.dp, end = 36.dp, top = 10.dp),
        ) {
            repeat(3) { i ->
                Box(
                    Modifier.weight(1f).height(2.dp)
                        .background(if (i < current - 1) Primary else Line),
                )
            }
        }
        Row(Modifier.fillMaxWidth()) {
            labels.forEachIndexed { i, label ->
                val idx = i + 1
                Column(Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                    StepNode(idx, current)
                    Spacer(Modifier.height(4.dp))
                    Text(
                        label, fontSize = 10.5.sp, textAlign = TextAlign.Center,
                        color = if (idx == current) Primary else Muted,
                        fontWeight = if (idx == current) FontWeight.W600 else FontWeight.Normal,
                    )
                }
            }
        }
    }
}

@Composable
private fun StepNode(idx: Int, current: Int) {
    val done = idx < current
    val active = idx == current
    Box(
        Modifier.size(20.dp)
            .background(if (done || active) Primary else Color.White, CircleShape)
            .border(1.dp, if (done || active) Primary else Line, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        if (done) {
            Icon(Icons.Filled.Check, contentDescription = null, tint = Color.White, modifier = Modifier.size(12.dp))
        } else {
            Text(
                "$idx", fontSize = 10.sp,
                color = if (active) Color.White else Muted,
                fontWeight = FontWeight.W600,
            )
        }
    }
}

/** 浅底圆角图标瓦片(对齐 user 端 IconTile)。 */
@Composable
private fun IconTile(icon: ImageVector, tint: Color, size: androidx.compose.ui.unit.Dp) {
    Box(
        Modifier.size(size).background(tint.copy(alpha = 0.12f), RoundedCornerShape(12.dp)),
        contentAlignment = Alignment.Center,
    ) { Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(size * 0.55f)) }
}
