package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import org.json.JSONObject
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted

// 工单列表行(对齐 home.html/orders.html 的 cell):no · 类型 / 地址 · 距离 / 状态标签 + 环节
@Composable
fun TicketCell(item: JSONObject, onClick: (() -> Unit)? = null,
               rightExtra: (@Composable () -> Unit)? = null) {
    val no = item.optString("ticketNo")
    val dist = if (item.isNull("distanceKm")) "" else " · 距您 ${item.optDouble("distanceKm")}km"
    Cell(
        title = "$no · ${item.optString("typeLabel")}",
        desc = item.optString("address") + dist,
        onClick = onClick,
        right = {
            Column(horizontalAlignment = Alignment.End) {
            StatusTag(item.optString("statusLabel"), item.optString("status"))
            if (!item.isNull("stageTotal")) {
                Text("${item.optInt("stage", -1)}/${item.optInt("stageTotal")} 环节",
                    fontSize = 12.sp, color = Muted)
            }
                rightExtra?.invoke()
            }
        },
    )
}
