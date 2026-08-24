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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.HourglassEmpty
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 对应 design/order-detail-v1.png:状态头 + 信息卡 + 4 里程碑 + 预计上门条 + 12 环节时间线 + 动作栏。
// 本文件只保留 Screen 编排与状态头;信息卡/进度卡/时间线/操作栏拆见同包 OrderInfoCard/OrderProgressCards/OrderTimeline/OrderActions。
@Composable
fun OrderScreen(nav: Nav, no: String) {
    var detail by remember { mutableStateOf<JSONObject?>(null) }
    var wrapper by remember { mutableStateOf<JSONObject?>(null) }
    var timeline by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    var showCancel by remember { mutableStateOf(false) }
    var notFound by remember { mutableStateOf(false) }
    LaunchedEffect(no, nav.refreshTick) {
        try {
            val resp = OrderApi.detail(no)
            wrapper = resp
            detail = resp.optJSONObject("order")
            timeline = resp.optJSONArray("timeline").toObjList()
            notFound = false
            err = ""
        } catch (e: Api.HttpError) {
            if (e.status == 404) notFound = true else err = "订单详情加载失败,请稍后重试"
        } catch (e: Exception) {
            err = "订单详情加载失败,请稍后重试"
        }
    }

    if (notFound) {
        Column(Modifier.fillMaxSize()) {
            TopBar("订单详情", onBack = { nav.pop() })
            EmptyState("订单不存在或已删除")
        }
        return
    }

    val order = detail
    val status = order?.optString("status") ?: ""
    // 底部固定操作栏高度约 84dp(内 padding 12+12 + Button 44dp),留 96dp 给滚动区尾,避免最后一行被遮。
    Box(Modifier.fillMaxSize().background(Palette.bg)) {
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()),
        ) {
            TopBar(
                title = "订单详情",
                onBack = { nav.pop() },
                action = if (status in setOf("PENDING", "RESERVED", "INSTALLING")) "取消" else null,
                onAction = { showCancel = true },
            )
            if (err.isNotEmpty()) Notice(err, Palette.err)
            StatusHeader(order)
            OrderInfoCard(order, wrapper)
            // 已取消订单:无装维动作,里程碑 + 12 环节时间线均隐藏;改展示 CancelInfoCard 给出取消阶段说明。
            if (status != "CANCELLED") {
                MilestoneBlock(order)
                if (status == "INSTALLING") EstimateBanner(order)
                OrderTimelineCard(timeline)
            } else {
                CancelInfoCard(order)
            }
            Spacer(Modifier.height(96.dp))
        }
        if (status.isNotEmpty()) FloatingActionBar(nav, no, order, status)
    }
    if (showCancel) CancelDialog(nav, no) { showCancel = false }
}

@Composable
private fun StatusHeader(order: JSONObject?) {
    com.ymm.boss.user.ui.AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconTile(statusIcon(order), statusTint(order), size = 48.dp, corner = 12.dp)
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    order?.optString("statusLabel").orDefault(order?.optString("status").labelOf()),
                    fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Palette.ink,
                )
                val desc = order?.optString("statusDesc").orEmpty()
                if (desc.isNotBlank()) {
                    Text(desc, fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 4.dp))
                }
            }
            Spacer(Modifier.width(12.dp))
            Column(horizontalAlignment = Alignment.End) {
                Text("订单号", fontSize = 11.sp, color = Palette.subtle)
                Text(
                    order?.optString("orderNo").orDefault("—"),
                    fontSize = 13.sp, fontWeight = FontWeight.W600, color = Palette.ink,
                    modifier = Modifier.padding(top = 2.dp),
                )
            }
        }
    }
}

private fun statusIcon(order: JSONObject?): ImageVector = when (order?.optString("status")) {
    "DONE" -> Icons.Filled.CheckCircle
    "INSTALLING" -> Icons.Filled.Build
    "RESERVED" -> Icons.Filled.HourglassEmpty
    else -> Icons.Filled.HourglassEmpty
}

private fun statusTint(order: JSONObject?): Color = when (order?.optString("status")) {
    "DONE" -> Palette.success
    "INSTALLING" -> Palette.primary
    "RESERVED" -> Palette.warn
    else -> Palette.muted
}

internal fun String?.labelOf(): String = when (this) {
    "PENDING" -> "待核查"
    "RESERVED" -> "已预占"
    "INSTALLING" -> "装维中"
    "DONE" -> "已完成"
    "CANCELLED" -> "已取消"
    else -> "—"
}

internal fun String?.orDefault(fallback: String): String =
    if (this.isNullOrBlank()) fallback else this
