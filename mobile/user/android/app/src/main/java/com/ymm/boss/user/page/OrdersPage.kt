package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 对应草稿 docs/user/orders.html:订单列表 + 状态筛选。tab 页。
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
        TopBar("我的订单")
        StatusSeg(filter) { filter = it }
        LazyColumn {
            item { if (err.isNotEmpty()) Notice(err, Palette.err) }
            if (orders.isEmpty() && err.isEmpty()) item {
                Text("暂无订单", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.fillMaxWidth().padding(top = 20.dp), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
            }
            items(orders) { o -> OrderCard(o, nav) }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }
}

@Composable
private fun StatusSeg(current: String, onSelect: (String) -> Unit) {
    val tabs = listOf(
        "all" to "全部", "in_progress" to "进行中", "done" to "已完成", "cancelled" to "已取消",
    )
    Row(
        Modifier.padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(18.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        tabs.forEach { (k, label) ->
            val active = k == current
            Text(
                label, fontSize = 13.5.sp,
                color = if (active) Palette.primary else Palette.muted,
                fontWeight = if (active) FontWeight.Bold else FontWeight.Normal,
                modifier = Modifier.clickable { onSelect(k) },
            )
        }
    }
}

private fun statusColor(status: String) = when (status) {
    "INSTALLING", "PENDING" -> Palette.primary
    "RESERVED" -> Palette.orange
    "DONE" -> Palette.success
    else -> Palette.muted
}

@Composable
private fun OrderCard(o: JSONObject, nav: Nav) {
    val no = o.optString("orderNo")
    AppCard(Modifier.clickable { nav.push(Route.Order(no)) }) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(no, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
            Spacer(Modifier.weight(1f))
            Tag(o.optString("statusLabel"), statusColor(o.optString("status")))
        }
        Notice("${o.optString("productName")} · ${o.optString("address")}")
        Text(
            stageLine(o), fontSize = 12.sp, color = Palette.muted,
        )
        if (o.optBoolean("canRate")) RateEntry(nav, no)
    }
}

private fun stageLine(o: JSONObject): String {
    val sb = StringBuilder("当前环节:${o.optString("stageLabel")}(${o.optString("stage")}/12)")
    when (o.optString("status")) {
        "DONE" -> sb.append(" · 全程 12 环节完成")
        else -> if (o.optString("estimateFinish").isNotEmpty()) sb.append(" · 预计 ").append(o.optString("estimateFinish"))
    }
    return sb.toString()
}

@Composable
private fun RateEntry(nav: Nav, no: String) {
    Text(
        "去评价", fontSize = 12.5.sp, color = Palette.primary,
        modifier = Modifier.padding(top = 8.dp).clickable { nav.push(Route.Rate(no)) },
    )
}
