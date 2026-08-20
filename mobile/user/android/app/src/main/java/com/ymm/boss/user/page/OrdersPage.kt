package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TabHeader
import org.json.JSONObject

// 对应设计稿 user-products-orders-profile.png 中屏(账单 tab):状态胶囊 + 订单卡 + 进度步骤条。
@Composable
fun OrdersScreen(nav: Nav) {
    var filter by remember { mutableStateOf("all") }
    var orders by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    LaunchedEffect(filter) {
        try {
            orders = OrderApi.list(filter).optJSONArray("items").toObjList()
            err = ""
        } catch (e: Exception) { err = "订单加载失败,请稍后重试" }
    }

    Column(Modifier.fillMaxSize()) {
        TabHeader("我的订单")
        StatusSeg(filter) { filter = it }
        LazyColumn {
            item { if (err.isNotEmpty()) Notice(err, Palette.err) }
            if (orders.isEmpty() && err.isEmpty()) item {
                Text(
                    "暂无订单", fontSize = 12.5.sp, color = Palette.muted, textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().padding(top = 20.dp),
                )
            }
            items(orders) { o -> OrderCard(o, nav) }
            if (orders.isNotEmpty()) item {
                Text(
                    "没有更多订单了", fontSize = 12.sp, color = Palette.subtle, textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
                )
            }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }
}

@Composable
private fun StatusSeg(current: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf("all" to "全部", "in_progress" to "进行中", "done" to "已完成", "cancelled" to "已取消")
            .forEach { (k, label) ->
                PillTab(label, active = k == current, onClick = { onSelect(k) }, plain = true)
            }
    }
}

private fun statusColor(status: String) = when (status) {
    "INSTALLING", "PENDING" -> Palette.primary
    "RESERVED" -> Palette.orange
    "DONE" -> Palette.success
    else -> Palette.muted
}

/** 12 环节 → 设计稿 4 里程碑:1 下单 / 2-7 受理 / 8-11 装维 / 12 完成。 */
private fun milestoneOf(stage: Int) = when {
    stage <= 1 -> 1
    stage <= 7 -> 2
    stage <= 11 -> 3
    else -> 4
}

@Composable
private fun OrderCard(o: JSONObject, nav: Nav) {
    val no = o.optString("orderNo")
    val status = o.optString("status")
    AppCard(Modifier.clickable { nav.push(Route.Order(no)) }) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(no, fontSize = 14.sp, fontWeight = FontWeight.W600, color = Palette.ink)
            Text(o.optString("createdAt"), fontSize = 12.sp, color = Palette.subtle)
        }
        Spacer(Modifier.height(10.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconTile(Icons.Filled.Home, Palette.primary, size = 48.dp, corner = 12.dp)
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Text(o.optString("productName"), fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink)
                    Text(o.optString("statusLabel"), fontSize = 13.sp, color = statusColor(status))
                }
                Spacer(Modifier.height(6.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Filled.LocationOn, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(14.dp))
                    Spacer(Modifier.width(4.dp))
                    Text(o.optString("address"), fontSize = 12.sp, color = Palette.muted, maxLines = 1)
                }
            }
        }
        when (status) {
            "DONE" -> DoneFooter(o, nav, no)
            "CANCELLED" -> Unit
            else -> ProgressBody(o)
        }
    }
}

@Composable
private fun ProgressBody(o: JSONObject) {
    Spacer(Modifier.height(14.dp))
    OrderStepper(milestoneOf(o.optInt("stage", 1)))
    if (o.optString("estimateFinish").isNotEmpty()) {
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier.fillMaxWidth().background(Color(0xFFF0F6FF), RoundedCornerShape(8.dp)).padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Filled.Person, contentDescription = null, tint = Palette.primary, modifier = Modifier.size(16.dp))
            Spacer(Modifier.width(8.dp))
            Text(
                "预计上门:${o.optString("estimateFinish")}", fontSize = 12.5.sp,
                color = Palette.ink, modifier = Modifier.weight(1f),
            )
            Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(18.dp))
        }
    }
}

@Composable
private fun DoneFooter(o: JSONObject, nav: Nav, no: String) {
    Spacer(Modifier.height(12.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(Icons.Filled.CheckCircle, contentDescription = null, tint = Palette.success, modifier = Modifier.size(20.dp))
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            Text("订单已完成", fontSize = 14.sp, fontWeight = FontWeight.W600, color = Palette.ink)
            Text("安装完成,感谢您的选择!", fontSize = 12.sp, color = Palette.muted)
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(18.dp))
    }
    if (o.optBoolean("canRate")) {
        Text(
            "去评价", fontSize = 12.5.sp, color = Palette.primary,
            modifier = Modifier.padding(top = 8.dp).clickable { nav.push(Route.Rate(no)) },
        )
    }
}

@Composable
private fun OrderStepper(current: Int) {
    val labels = listOf("提交订单", "受理成功", "上门安装", "完成")
    Box(Modifier.fillMaxWidth().height(52.dp)) {
        Row(
            Modifier.fillMaxWidth().padding(start = 36.dp, end = 36.dp, top = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(0.dp),
        ) {
            repeat(3) { i ->
                Box(
                    Modifier.weight(1f).height(2.dp)
                        .background(if (i < current - 1) Palette.primary else Palette.line),
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
                        color = if (idx == current) Palette.primary else Palette.muted,
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
            .background(
                if (done || active) Palette.primary else Palette.panel, CircleShape,
            )
            .border(1.dp, if (done || active) Palette.primary else Palette.line, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        if (done) {
            Icon(Icons.Filled.Check, contentDescription = null, tint = Color.White, modifier = Modifier.size(12.dp))
        } else {
            Text(
                "$idx", fontSize = 10.sp,
                color = if (active) Color.White else Palette.subtle,
                fontWeight = FontWeight.W600,
            )
        }
    }
}
