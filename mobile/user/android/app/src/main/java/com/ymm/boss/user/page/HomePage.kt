package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.ui.draw.shadow
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.Canvas
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.List
import androidx.compose.material.icons.outlined.NotificationsNone
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.CreditCard
import androidx.compose.material.icons.outlined.SignalCellularAlt
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.HeadsetMic
import androidx.compose.material.icons.outlined.Assignment
import androidx.compose.material.icons.outlined.Router
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.*
import org.json.JSONObject

private data class HomeData(
    val name: String = "张先生", val phone: String = "138****8888", val plan: String = "家庭宽带 1000M",
    val bill: String = "89.00", val balance: String = "120.00", val end: String = "2026-12-31",
    val orders: List<JSONObject> = listOf(JSONObject().apply { put("orderNo", "ORD-20260818001"); put("productName", "家庭宽带 1000M"); put("address", "广东省深圳市南山区科技园"); put("stage", 8) }),
    val services: List<JSONObject> = listOf(JSONObject().apply { put("name", "家庭宽带 1000M"); put("desc", "服务在线 · 网络正常") }),
)

@Composable fun HomeScreen(nav: Nav) {
    var data by remember { mutableStateOf(HomeData()) }
    var loading by remember { mutableStateOf(true) }
    LaunchedEffect(Unit) { runCatching { UserApi.misc.home() }.onSuccess { d -> data = data.copy(name = d.optString("customerName", data.name), phone = d.optString("phoneMasked", data.phone), plan = d.optJSONObject("plan")?.optString("name") ?: data.plan, bill = d.optString("currentBill", data.bill), balance = d.optString("balance", data.balance), end = d.optString("contractEnd", data.end)) }; loading = false }
    Box(Modifier.fillMaxSize().background(Palette.bg)) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) { Header(data, nav); Plan(data, nav); Quick(nav); Orders(data, nav); Services(data); Spacer(Modifier.height(16.dp)) }
        if (loading) CircularProgressIndicator(Modifier.size(20.dp).align(Alignment.TopCenter).padding(top = 9.dp), color = Color.White, strokeWidth = 2.dp)
    }
}

@Composable private fun Header(d: HomeData, nav: Nav) {
    Box(Modifier.fillMaxWidth().height(120.dp).background(Brush.linearGradient(listOf(Palette.primary, Palette.primary2)))) {
        Canvas(Modifier.fillMaxSize()) {
            val center = androidx.compose.ui.geometry.Offset(size.width * .84f, size.height * .88f)
            listOf(.26f, .36f, .47f).forEach { radius -> drawCircle(Color.White.copy(alpha = .14f), size.minDimension * radius, center, style = Stroke(2.dp.toPx())) }
        }
        Column(Modifier.fillMaxWidth().padding(start = 24.dp, end = 24.dp, top = 12.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) { Column(Modifier.weight(1f)) { Text("早上好，${d.name}", color = Color.White, fontSize = 21.sp, fontWeight = FontWeight.Bold); Text(d.phone, color = Color.White, fontSize = 14.sp, modifier = Modifier.padding(top = 7.dp)) }; Box(Modifier.size(38.dp).clickable { nav.push(Route.Messages) }) { Icon(Icons.Outlined.List, "消息", tint = Color.White, modifier = Modifier.fillMaxSize().padding(3.dp)); Box(Modifier.size(10.dp).background(Palette.err, CircleShape).align(Alignment.TopEnd)) } }
        Row(Modifier.padding(top = 3.dp).background(Color.White.copy(.15f), RoundedCornerShape(22.dp)).padding(horizontal = 14.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) { StatusDot(); Text("  服务在线 · 网络正常", color = Color.White, fontSize = 13.sp) }
        }
    }
}

@Composable private fun Plan(d: HomeData, nav: Nav) {
    Column(Modifier.padding(horizontal = 14.dp, vertical = 10.dp).offset(y = (-22).dp).shadow(4.dp, RoundedCornerShape(16.dp)).background(Palette.panel, RoundedCornerShape(16.dp)).padding(10.dp)) {
        Row(Modifier.fillMaxWidth().clickable { nav.push(Route.MyPlan) }, verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.Home, null, tint = Palette.primary, modifier = Modifier.size(44.dp).background(Color(0xFFEFF5FF), CircleShape).padding(10.dp)); Text(d.plan, fontSize = 17.sp, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f).padding(start = 10.dp)); Tag("在网", Palette.success); Text("›", fontSize = 25.sp, color = Palette.muted) }
        Row(Modifier.fillMaxWidth().padding(top = 14.dp), horizontalArrangement = Arrangement.SpaceEvenly) { Metric("本月账单", "¥ ${d.bill}", Palette.primary) { nav.push(Route.Bills) }; Divider(); Metric("套餐余额", "¥ ${d.balance}", Palette.primary); Divider(); Metric("合约到期", d.end, Palette.ink) }
    }
}
@Composable private fun Metric(a: String, b: String, c: Color, click: (() -> Unit)? = null) { Column(Modifier.clickable(enabled = click != null) { click?.invoke() }, horizontalAlignment = Alignment.CenterHorizontally) { Text(a, fontSize = 12.sp); Text(b, fontSize = 18.sp, fontWeight = FontWeight.Bold, color = c, modifier = Modifier.padding(top = 4.dp)) } }
@Composable private fun Divider() { Spacer(Modifier.width(1.dp).height(42.dp).background(Palette.line)) }

@Composable private fun Quick(nav: Nav) {
    val items = listOf("办套餐" to Route.Products, "查订单" to Route.Orders, "缴费用" to Route.Pay, "报故障" to Route.Fault, "充值" to Route.Topup, "查用量" to Route.Usage, "消息" to Route.Messages, "客服" to Route.Service)
    Column(Modifier.padding(horizontal = 14.dp, vertical = 4.dp).shadow(3.dp, RoundedCornerShape(16.dp)).background(Palette.panel, RoundedCornerShape(16.dp)).padding(vertical = 6.dp)) { items.chunked(4).forEachIndexed { index, row -> Row(Modifier.fillMaxWidth().height(68.dp)) { row.forEachIndexed { column, (label, route) -> Column(Modifier.weight(1f).fillMaxHeight().clickable { nav.push(route) }.padding(horizontal = if (column == 0) 0.dp else 1.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) { Icon(when (label) { "办套餐" -> Icons.Outlined.ShoppingBag; "查订单" -> Icons.Outlined.Assignment; "缴费用" -> Icons.Outlined.Payments; "报故障" -> Icons.Outlined.Build; "充值" -> Icons.Outlined.CreditCard; "查用量" -> Icons.Outlined.SignalCellularAlt; "消息" -> Icons.Outlined.ChatBubbleOutline; else -> Icons.Outlined.HeadsetMic }, label, tint = when (label) { "查订单", "查用量" -> Palette.success; "缴费用", "消息" -> Palette.orange; "报故障", "客服" -> Palette.purple; else -> Palette.primary }, modifier = Modifier.size(32.dp)); Text(label, fontSize = 11.sp, modifier = Modifier.padding(top = 3.dp)) } } }; if (index == 0) Spacer(Modifier.fillMaxWidth().height(1.dp).background(Palette.line)) } }
}

@Composable private fun Orders(d: HomeData, nav: Nav) {
    Card {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) { Text("进行中订单", fontSize = 20.sp, fontWeight = FontWeight.Bold); Text("全部 〉", color = Palette.primary, modifier = Modifier.clickable { nav.push(Route.Orders) }) }
        d.orders.firstOrNull()?.let { o ->
            Row(Modifier.padding(top = 18.dp), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.Assignment, null, tint = Palette.primary, modifier = Modifier.size(58.dp).background(Color(0xFFEFF5FF), RoundedCornerShape(18.dp)).padding(14.dp))
                Column(Modifier.padding(start = 16.dp).weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) { Text("${o.optString("orderNo")} · ${o.optString("productName")}", fontSize = 13.sp, maxLines = 1, modifier = Modifier.weight(1f)); Tag("装维中", Palette.primary) }
                    Row(Modifier.padding(top = 7.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.LocationOn, null, tint = Palette.muted, modifier = Modifier.size(16.dp)); Text(o.optString("address"), color = Palette.muted, fontSize = 12.sp, modifier = Modifier.padding(start = 4.dp)) }
                    Row(Modifier.padding(top = 12.dp), verticalAlignment = Alignment.CenterVertically) { Text("上门安装", color = Palette.primary, fontSize = 14.sp); Text("  ${o.optInt("stage", 8)}/12", color = Palette.muted, fontSize = 12.sp); Spacer(Modifier.width(10.dp)); Box(Modifier.weight(1f).height(7.dp).background(Palette.line, RoundedCornerShape(5.dp))) { Box(Modifier.fillMaxWidth(o.optInt("stage", 8) / 12f).height(7.dp).background(Palette.primary, RoundedCornerShape(5.dp))) } }
                }
                Text("〉", color = Palette.muted, fontSize = 26.sp, modifier = Modifier.padding(start = 6.dp))
            }
        }
    }
}

@Composable private fun Services(d: HomeData) {
    Card {
        Text("我的服务", fontSize = 20.sp, fontWeight = FontWeight.Bold)
        d.services.firstOrNull()?.let { s ->
            Row(Modifier.padding(top = 17.dp), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.Router, null, tint = Palette.primary, modifier = Modifier.size(70.dp).background(Color(0xFFEFF5FF), RoundedCornerShape(22.dp)).padding(15.dp))
                Column(Modifier.padding(start = 16.dp).weight(1f)) { Row(verticalAlignment = Alignment.CenterVertically) { Text(s.optString("name"), fontSize = 17.sp, modifier = Modifier.weight(1f)); Tag("在网", Palette.success) }; Row(Modifier.padding(top = 8.dp), verticalAlignment = Alignment.CenterVertically) { StatusDot(); Text("  ${s.optString("desc")}", color = Palette.success, fontSize = 14.sp) } }
                Text("〉", color = Palette.muted, fontSize = 26.sp)
            }
        }
    }
}

@Composable private fun Card(content: @Composable ColumnScope.() -> Unit) { Column(Modifier.padding(horizontal = 14.dp, vertical = 6.dp).shadow(3.dp, RoundedCornerShape(16.dp)).background(Palette.panel, RoundedCornerShape(16.dp)).padding(18.dp), content = content) }
