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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Assignment
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.CreditCard
import androidx.compose.material.icons.outlined.HeadsetMic
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.NotificationsNone
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Router
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material.icons.outlined.SignalCellularAlt
import androidx.compose.material3.CircularProgressIndicator
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
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.StatusDot
import com.ymm.boss.user.ui.Tag
import org.json.JSONObject

private data class HomeData(
    val customerName: String = "张先生", val phoneMasked: String = "138****8888", val onlineStatus: String = "服务在线 · 网络正常",
    val planName: String = "家庭宽带 1000M", val currentBill: String = "89.00", val balance: String = "120.00", val contractEnd: String = "2026-12-31",
    val hasUnread: Boolean = true,
    val ongoingOrders: List<JSONObject> = listOf(JSONObject().apply { put("orderNo", "ORD-20260818001"); put("productName", "家庭宽带 1000M"); put("address", "广东省深圳市南山区科技园"); put("statusLabel", "装维中"); put("stageLabel", "上门安装"); put("stage", 8) }),
    val services: List<JSONObject> = listOf(JSONObject().apply { put("name", "家庭宽带 1000M"); put("desc", "服务在线 · 网络正常"); put("statusLabel", "在网") }),
)

@Composable
fun HomeScreen(nav: Nav) {
    var data by remember { mutableStateOf(HomeData()) }
    var loading by remember { mutableStateOf(true) }
    suspend fun reload() {
        loading = true
        runCatching { UserApi.misc.home() }.onSuccess { d -> data = data.copy(customerName = d.optString("customerName", data.customerName), phoneMasked = d.optString("phoneMasked", data.phoneMasked), onlineStatus = d.optString("onlineStatus", data.onlineStatus), planName = d.optJSONObject("plan")?.optString("name") ?: data.planName, currentBill = d.optString("currentBill", data.currentBill), balance = d.optString("balance", data.balance), contractEnd = d.optString("contractEnd", data.contractEnd), hasUnread = d.optBoolean("hasUnread", data.hasUnread), ongoingOrders = d.optJSONArray("ongoingOrders")?.toList()?.ifEmpty { data.ongoingOrders } ?: data.ongoingOrders, services = d.optJSONArray("services")?.toList()?.ifEmpty { data.services } ?: data.services) }
        loading = false
    }
    LaunchedEffect(Unit) { reload() }
    Box(Modifier.fillMaxSize().background(Palette.bg)) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) { Header(data, nav); PlanCard(data, nav); QuickGrid(nav); OrdersCard(data, nav); ServicesCard(data); Spacer(Modifier.height(16.dp)) }
        if (loading) CircularProgressIndicator(Modifier.size(22.dp).align(Alignment.TopCenter).padding(top = 10.dp), color = Color.White, strokeWidth = 2.dp)
    }
}

@Composable private fun Header(data: HomeData, nav: Nav) {
    Column(Modifier.fillMaxWidth().height(300.dp).background(Brush.linearGradient(listOf(Palette.primary, Palette.primary2))).padding(horizontal = 24.dp, vertical = 20.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) { Column(Modifier.weight(1f)) { Text("早上好，${data.customerName}", color = Color.White, fontSize = 25.sp, fontWeight = FontWeight.Bold); Text(data.phoneMasked, color = Color.White, fontSize = 16.sp, modifier = Modifier.padding(top = 9.dp)) }; Box(Modifier.size(42.dp).clickable { nav.push(Route.Messages) }) { Icon(Icons.Outlined.NotificationsNone, "消息", tint = Color.White, modifier = Modifier.fillMaxSize().padding(3.dp)); if (data.hasUnread) Box(Modifier.size(11.dp).background(Palette.err, CircleShape).align(Alignment.TopEnd)) } }
        Row(Modifier.padding(top = 17.dp).background(Color.White.copy(.15f), RoundedCornerShape(22.dp)).padding(horizontal = 16.dp, vertical = 9.dp), verticalAlignment = Alignment.CenterVertically) { StatusDot(); Spacer(Modifier.width(9.dp)); Text(data.onlineStatus, color = Color.White, fontSize = 15.sp) }
    }
}

@Composable private fun PlanCard(data: HomeData, nav: Nav) {
    Column(Modifier.padding(horizontal = 14.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(horizontal = 18.dp, vertical = 17.dp)) {
        Row(Modifier.fillMaxWidth().clickable { nav.push(Route.MyPlan) }, verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.Home, null, tint = Palette.primary, modifier = Modifier.size(54.dp).background(Color(0xFFEFF5FF), CircleShape).padding(13.dp)); Text(data.planName, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Palette.ink, modifier = Modifier.weight(1f).padding(start = 14.dp)); Tag("在网", Palette.success); Text("›", color = Palette.muted, fontSize = 29.sp, modifier = Modifier.padding(start = 12.dp)) }
        Row(Modifier.fillMaxWidth().padding(top = 22.dp), horizontalArrangement = Arrangement.SpaceEvenly) { Metric("本月账单", "¥ ${data.currentBill}", Palette.primary) { nav.push(Route.Bills) }; Metric("套餐余额", "¥ ${data.balance}", Palette.primary) { nav.push(Route.Topup) }; Metric("合约到期", data.contractEnd, Palette.ink) }
    }
}

@Composable private fun Metric(label: String, value: String, color: Color, onClick: (() -> Unit)? = null) { Column(Modifier.clickable(enabled = onClick != null) { onClick?.invoke() }, horizontalAlignment = Alignment.CenterHorizontally) { Text(label, fontSize = 14.sp, color = Palette.ink); Text(value, fontSize = 23.sp, fontWeight = FontWeight.Bold, color = color, modifier = Modifier.padding(top = 7.dp)) } }

private data class Quick(val label: String, val icon: androidx.compose.ui.graphics.vector.ImageVector, val color: Color, val route: Route)
@Composable private fun QuickGrid(nav: Nav) {
    val entries = listOf(Quick("办套餐", Icons.Outlined.ShoppingBag, Palette.primary, Route.Products), Quick("查订单", Icons.Outlined.Assignment, Palette.success, Route.Orders), Quick("缴费用", Icons.Outlined.Payments, Palette.orange, Route.Pay), Quick("报故障", Icons.Outlined.Build, Palette.purple, Route.Fault), Quick("充值", Icons.Outlined.CreditCard, Palette.primary, Route.Topup), Quick("查用量", Icons.Outlined.SignalCellularAlt, Palette.success, Route.Usage), Quick("消息", Icons.Outlined.ChatBubbleOutline, Palette.orange, Route.Messages), Quick("客服", Icons.Outlined.HeadsetMic, Palette.purple, Route.Service))
    Column(Modifier.padding(horizontal = 14.dp, vertical = 10.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(vertical = 11.dp)) { entries.chunked(4).forEachIndexed { index, row -> Row(Modifier.fillMaxWidth().height(94.dp)) { row.forEach { e -> Column(Modifier.weight(1f).fillMaxSize().clickable { nav.push(e.route) }, horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) { Icon(e.icon, e.label, tint = e.color, modifier = Modifier.size(45.dp)); Text(e.label, fontSize = 15.sp, color = Palette.ink, modifier = Modifier.padding(top = 8.dp)) } } }; if (index == 0) Spacer(Modifier.fillMaxWidth().height(1.dp).background(Palette.line)) } }
}

@Composable private fun OrdersCard(data: HomeData, nav: Nav) {
    Column(Modifier.padding(horizontal = 14.dp, vertical = 1.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(horizontal = 18.dp, vertical = 17.dp)) { Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) { Text("进行中订单", fontSize = 22.sp, fontWeight = FontWeight.Bold, color = Palette.ink); Text("全部 ›", fontSize = 15.sp, color = Palette.primary, modifier = Modifier.clickable { nav.push(Route.Orders) }) }; data.ongoingOrders.firstOrNull()?.let { o -> Row(Modifier.padding(top = 18.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.Assignment, null, tint = Palette.primary, modifier = Modifier.size(70.dp).background(Color(0xFFEFF5FF), RoundedCornerShape(22.dp)).padding(17.dp)); Column(Modifier.padding(start = 16.dp).weight(1f)) { Row(verticalAlignment = Alignment.CenterVertically) { Text("${o.optString("orderNo")} · ${o.optString("productName")}", fontSize = 16.sp, fontWeight = FontWeight.Medium, color = Palette.ink, modifier = Modifier.weight(1f)); Tag(o.optString("statusLabel", "装维中"), Palette.primary) }; Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 7.dp)) { Icon(Icons.Outlined.LocationOn, null, tint = Palette.muted, modifier = Modifier.size(17.dp)); Text(o.optString("address"), color = Palette.muted, fontSize = 13.sp, modifier = Modifier.padding(start = 4.dp)) }; Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 13.dp)) { Text(o.optString("stageLabel", "上门安装"), color = Palette.primary, fontSize = 15.sp); Text("  ${o.optInt("stage", 8)}/12", color = Palette.muted, fontSize = 13.sp); Spacer(Modifier.width(12.dp)); Box(Modifier.weight(1f).height(7.dp).background(Palette.line, RoundedCornerShape(5.dp))) { Box(Modifier.fillMaxWidth(o.optInt("stage", 8) / 12f).height(7.dp).background(Palette.primary, RoundedCornerShape(5.dp))) } } } } } }
    }
}

@Composable private fun ServicesCard(data: HomeData) {
    Column(Modifier.padding(horizontal = 14.dp, vertical = 10.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(horizontal = 18.dp, vertical = 17.dp)) { Text("我的服务", fontSize = 22.sp, fontWeight = FontWeight.Bold, color = Palette.ink); data.services.firstOrNull()?.let { s -> Row(Modifier.padding(top = 17.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.Router, null, tint = Palette.primary, modifier = Modifier.size(70.dp).background(Color(0xFFEFF5FF), RoundedCornerShape(22.dp)).padding(15.dp)); Column(Modifier.padding(start = 16.dp).weight(1f)) { Row(verticalAlignment = Alignment.CenterVertically) { Text(s.optString("name"), fontSize = 18.sp, fontWeight = FontWeight.Medium, modifier = Modifier.weight(1f)); Tag(s.optString("statusLabel", "在网"), Palette.success) }; Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 8.dp)) { StatusDot(); Text(s.optString("desc", data.onlineStatus), color = Palette.success, fontSize = 15.sp, modifier = Modifier.padding(start = 8.dp)) } }; Text("›", color = Palette.muted, fontSize = 29.sp) } } }
