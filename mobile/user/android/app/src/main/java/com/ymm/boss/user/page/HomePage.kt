package com.ymm.boss.user.page

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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.List
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.ShoppingCart
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
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.StatusDot
import com.ymm.boss.user.ui.Tag
import org.json.JSONArray
import org.json.JSONObject

/** 对应草稿 docs/user/home.html:工作台聚合页。 */
private data class HomeData(
    val customerName: String = "张先生",
    val phoneMasked: String = "138****8888",
    val onlineStatus: String = "服务在线 · 网络正常",
    val planName: String = "家庭宽带 1000M",
    val currentBill: String = "89.00",
    val balance: String = "120.00",
    val contractEnd: String = "2026-12-31",
    val hasUnread: Boolean = true,
    val ongoingOrders: List<JSONObject> = listOf(JSONObject().apply {
        put("orderNo", "ORD-20260818001"); put("productName", "家庭宽带 1000M")
        put("address", "广东省深圳市南山区科技园"); put("statusLabel", "装维中")
        put("stageLabel", "上门安装"); put("stage", "8")
    }),
    val services: List<JSONObject> = listOf(JSONObject().apply {
        put("name", "家庭宽带 1000M"); put("desc", "服务在线 · 网络正常"); put("statusLabel", "在网")
    }),
)

@Composable
fun HomeScreen(nav: Nav) {
    var data by remember { mutableStateOf(HomeData()) }
    LaunchedEffect(Unit) {
        try {
            val d = UserApi.misc.home()
            data = HomeData(
                customerName = d.optString("customerName", data.customerName),
                phoneMasked = d.optString("phoneMasked", data.phoneMasked),
                onlineStatus = d.optString("onlineStatus", "服务在线 · 网络正常"),
                planName = d.optJSONObject("plan")?.optString("name") ?: data.planName,
                currentBill = d.optString("currentBill", data.currentBill),
                balance = d.optString("balance", data.balance),
                contractEnd = d.optString("contractEnd", data.contractEnd),
                hasUnread = d.optBoolean("hasUnread", data.hasUnread),
                ongoingOrders = d.optJSONArray("ongoingOrders")?.toList()?.ifEmpty { data.ongoingOrders } ?: data.ongoingOrders,
                services = d.optJSONArray("services")?.toList()?.ifEmpty { data.services } ?: data.services,
            )
        } catch (e: Exception) { /* 假数据不可达时保留骨架,与草稿一致 */ }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        HomeHead(data, nav)
        StatusCard(data, nav)
        QuickGrid(nav)
        AppCard {
            CardTitle("进行中订单", "全部") { nav.push(Route.Orders) }
            if (data.ongoingOrders.isEmpty()) EmptyHint("暂无进行中订单")
            data.ongoingOrders.forEach { o ->
                OrderCell(o) { nav.push(Route.Order(o.optString("orderNo"))) }
            }
        }
        AppCard {
            CardTitle("我的服务", "在线状态")
            if (data.services.isEmpty()) EmptyHint("暂无服务")
            data.services.forEach { s ->
                CellRow(
                    title = s.optString("name"),
                    desc = s.optString("desc"),
                    right = { Tag(s.optString("statusLabel", "在网"), Palette.success) },
                )
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun EmptyHint(text: String) {
    Text(text, fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
}

@Composable
private fun HomeHead(data: HomeData, nav: Nav) {
    Column(
        Modifier.fillMaxWidth()
            .background(Brush.linearGradient(listOf(Palette.primary, Palette.primary2)))
            .padding(start = 24.dp, end = 24.dp, top = 20.dp, bottom = 26.dp),
    ) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                Column {
                    Text("早上好，${data.customerName}", color = Color.White, fontSize = 25.sp, fontWeight = FontWeight.Bold)
                    Text(data.phoneMasked, color = Color.White.copy(alpha = 0.9f), fontSize = 16.sp, modifier = Modifier.padding(top = 8.dp))
                }
            }
            Box {
                Icon(Icons.Outlined.List, "消息", tint = Color.White, modifier = Modifier.size(34.dp).clickable { nav.push(Route.Messages) })
                if (data.hasUnread) Box(Modifier.size(8.dp).background(Palette.err, CircleShape).align(Alignment.TopEnd))
            }
        }
        Row(Modifier.padding(top = 16.dp).background(Color.White.copy(alpha = .14f), RoundedCornerShape(24.dp)).padding(horizontal = 16.dp, vertical = 9.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(9.dp)) {
            StatusDot()
            Text(data.onlineStatus, color = Color.White, fontSize = 15.sp)
        }
    }
}

@Composable
private fun StatusCard(data: HomeData, nav: Nav) {
    Column(
        Modifier
            .padding(horizontal = 14.dp, vertical = 12.dp)
            .background(Palette.panel, RoundedCornerShape(12.dp))
            .padding(16.dp)
            .fillMaxWidth(),
    ) {
        Row(Modifier.fillMaxWidth().padding(bottom = 12.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Outlined.Home, null, tint = Palette.primary, modifier = Modifier.size(42.dp).background(Color(0xFFEFF5FF), CircleShape).padding(10.dp))
            Text(data.planName, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Palette.ink, modifier = Modifier.weight(1f).padding(start = 14.dp))
            Tag("在网", Palette.success)
            Text("›", color = Palette.muted, fontSize = 26.sp, modifier = Modifier.padding(start = 14.dp))
        }
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
            NetItem("本月账单", "¥${data.currentBill}") { nav.push(Route.Bills) }
            NetItem("套餐余额", "¥${data.balance}") { nav.push(Route.Topup) }
            NetItem("合约到期", data.contractEnd)
        }
    }
}

@Composable
private fun NetItem(label: String, value: String, onClick: (() -> Unit)? = null) {
    Column(Modifier.clickable(enabled = onClick != null) { onClick?.invoke() }) {
        Text(label, fontSize = 12.sp, color = Palette.muted)
        Text(value, fontSize = 24.sp, fontWeight = FontWeight.Bold, color = if (label == "合约到期") Palette.ink else Palette.primary, modifier = Modifier.padding(top = 4.dp))
    }
}

@Composable
private fun OrderCell(o: JSONObject, onClick: () -> Unit) {
    CellRow(
        title = "${o.optString("orderNo")} · ${o.optString("productName")}",
        desc = o.optString("address"),
        onClick = onClick,
        right = {
            Column(horizontalAlignment = Alignment.End) {
                Tag(o.optString("statusLabel"), Palette.primary)
                Text(
                    "${o.optString("stageLabel")} ${o.optString("stage")}/12",
                    fontSize = 11.sp, color = Palette.muted, modifier = Modifier.padding(top = 4.dp),
                )
            }
        },
    )
}

private data class QuickEntry(val label: String, val icon: androidx.compose.ui.graphics.vector.ImageVector, val color: Color, val route: Route)

@Composable
private fun QuickGrid(nav: Nav) {
    val entries = listOf(
        QuickEntry("办套餐", Icons.Outlined.ShoppingCart, Palette.primary, Route.Products),
        QuickEntry("查订单", Icons.Outlined.List, Palette.success, Route.Orders),
        QuickEntry("缴费用", Icons.Outlined.ShoppingCart, Palette.orange, Route.Pay),
        QuickEntry("报故障", Icons.Outlined.Person, Palette.purple, Route.Fault),
        QuickEntry("充值", Icons.Outlined.ShoppingCart, Palette.primary, Route.Topup),
        QuickEntry("查用量", Icons.Outlined.List, Palette.success, Route.Usage),
        QuickEntry("消息", Icons.Outlined.List, Palette.orange, Route.Messages),
        QuickEntry("客服", Icons.Outlined.Person, Palette.purple, Route.Service),
    )
    Column(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 4.dp)) {
        entries.chunked(4).forEach { row ->
            Row(Modifier.fillMaxWidth()) {
                row.forEach { e ->
                    Column(
                        Modifier.weight(1f).padding(vertical = 6.dp).clickable { nav.push(e.route) },
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Icon(e.icon, e.label, tint = e.color, modifier = Modifier.size(44.dp).padding(5.dp))
                        Text(e.label, fontSize = 15.sp, color = Palette.ink, modifier = Modifier.padding(top = 7.dp))
                    }
                }
            }
        }
    }
}
