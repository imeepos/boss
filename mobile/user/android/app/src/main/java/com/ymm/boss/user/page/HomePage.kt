package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.List
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
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
    Column(Modifier.fillMaxWidth().height(300.dp).background(Brush.linearGradient(listOf(Palette.primary, Palette.primary2))).padding(24.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) { Column(Modifier.weight(1f)) { Text("早上好，${d.name}", color = Color.White, fontSize = 25.sp, fontWeight = FontWeight.Bold); Text(d.phone, color = Color.White, fontSize = 16.sp, modifier = Modifier.padding(top = 9.dp)) }; Box(Modifier.size(40.dp).clickable { nav.push(Route.Messages) }) { Icon(Icons.Outlined.List, "消息", tint = Color.White, modifier = Modifier.fillMaxSize().padding(3.dp)); Box(Modifier.size(11.dp).background(Palette.err, CircleShape).align(Alignment.TopEnd)) } }
        Row(Modifier.padding(top = 17.dp).background(Color.White.copy(.15f), RoundedCornerShape(22.dp)).padding(horizontal = 16.dp, vertical = 9.dp), verticalAlignment = Alignment.CenterVertically) { StatusDot(); Text("  服务在线 · 网络正常", color = Color.White, fontSize = 15.sp) }
    }
}

@Composable private fun Plan(d: HomeData, nav: Nav) {
    Column(Modifier.padding(horizontal = 14.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(18.dp)) {
        Row(Modifier.fillMaxWidth().clickable { nav.push(Route.MyPlan) }, verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.Home, null, tint = Palette.primary, modifier = Modifier.size(54.dp).background(Color(0xFFEFF5FF), CircleShape).padding(13.dp)); Text(d.plan, fontSize = 20.sp, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f).padding(start = 14.dp)); Tag("在网", Palette.success); Text("›", fontSize = 29.sp, color = Palette.muted) }
        Row(Modifier.fillMaxWidth().padding(top = 22.dp), horizontalArrangement = Arrangement.SpaceEvenly) { Metric("本月账单", "¥ ${d.bill}", Palette.primary) { nav.push(Route.Bills) }; Metric("套餐余额", "¥ ${d.balance}", Palette.primary); Metric("合约到期", d.end, Palette.ink) }
    }
}
@Composable private fun Metric(a: String, b: String, c: Color, click: (() -> Unit)? = null) { Column(Modifier.clickable(enabled = click != null) { click?.invoke() }, horizontalAlignment = Alignment.CenterHorizontally) { Text(a, fontSize = 14.sp); Text(b, fontSize = 23.sp, fontWeight = FontWeight.Bold, color = c, modifier = Modifier.padding(top = 7.dp)) } }

@Composable private fun Quick(nav: Nav) {
    val items = listOf("办套餐" to Route.Products, "查订单" to Route.Orders, "缴费用" to Route.Pay, "报故障" to Route.Fault, "充值" to Route.Topup, "查用量" to Route.Usage, "消息" to Route.Messages, "客服" to Route.Service)
    Column(Modifier.padding(14.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(vertical = 10.dp)) { items.chunked(4).forEach { row -> Row(Modifier.fillMaxWidth().height(94.dp)) { row.forEach { (label, route) -> Column(Modifier.weight(1f).fillMaxHeight().clickable { nav.push(route) }, horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) { Icon(if (label == "办套餐" || label == "缴费用" || label == "充值") Icons.Outlined.ShoppingCart else if (label == "客服" || label == "报故障") Icons.Outlined.Person else Icons.Outlined.List, label, tint = when (label) { "查订单" -> Palette.success; "缴费用", "消息" -> Palette.orange; "报故障", "客服" -> Palette.purple; else -> Palette.primary }, modifier = Modifier.size(43.dp)); Text(label, fontSize = 15.sp, modifier = Modifier.padding(top = 8.dp)) } } } } }
}

@Composable private fun Orders(d: HomeData, nav: Nav) { Card { Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) { Text("进行中订单", fontSize = 22.sp, fontWeight = FontWeight.Bold); Text("全部 ›", color = Palette.primary, modifier = Modifier.clickable { nav.push(Route.Orders) }) }; d.orders.firstOrNull()?.let { o -> Row(Modifier.padding(top = 18.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.List, null, tint = Palette.primary, modifier = Modifier.size(70.dp).background(Color(0xFFEFF5FF), RoundedCornerShape(22.dp)).padding(17.dp)); Column(Modifier.padding(start = 16.dp)) { Text("${o.optString("orderNo")} · ${o.optString("productName")}", fontSize = 16.sp); Text(o.optString("address"), color = Palette.muted, fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp)); Text("上门安装   ${o.optInt("stage", 8)}/12  ━━━━━", color = Palette.primary, modifier = Modifier.padding(top = 12.dp)) } } } } }
@Composable private fun Services(d: HomeData) { Card { Text("我的服务", fontSize = 22.sp, fontWeight = FontWeight.Bold); d.services.firstOrNull()?.let { s -> Row(Modifier.padding(top = 17.dp), verticalAlignment = Alignment.CenterVertically) { Icon(Icons.Outlined.Home, null, tint = Palette.primary, modifier = Modifier.size(70.dp).background(Color(0xFFEFF5FF), RoundedCornerShape(22.dp)).padding(15.dp)); Column(Modifier.padding(start = 16.dp)) { Text(s.optString("name"), fontSize = 18.sp); Row(Modifier.padding(top = 8.dp), verticalAlignment = Alignment.CenterVertically) { StatusDot(); Text("  ${s.optString("desc")}", color = Palette.success) } } } } } }
@Composable private fun Card(content: @Composable ColumnScope.() -> Unit) { Column(Modifier.padding(horizontal = 14.dp, vertical = 6.dp).background(Palette.panel, RoundedCornerShape(16.dp)).padding(18.dp), content = content) }
