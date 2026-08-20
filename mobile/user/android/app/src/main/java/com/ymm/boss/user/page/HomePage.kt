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
    val customerName: String = "加载中…",
    val phoneMasked: String = "",
    val onlineStatus: String = "服务在线 · 网络正常",
    val planName: String = "—",
    val currentBill: String = "—",
    val balance: String = "—",
    val contractEnd: String = "—",
    val hasUnread: Boolean = false,
    val ongoingOrders: List<JSONObject> = emptyList(),
    val services: List<JSONObject> = emptyList(),
)

@Composable
fun HomeScreen(nav: Nav) {
    var data by remember { mutableStateOf(HomeData()) }
    LaunchedEffect(Unit) {
        try {
            val d = UserApi.misc.home()
            data = HomeData(
                customerName = d.optString("customerName", "客户"),
                phoneMasked = d.optString("phoneMasked"),
                onlineStatus = d.optString("onlineStatus", "服务在线 · 网络正常"),
                planName = d.optJSONObject("plan")?.optString("name") ?: "—",
                currentBill = d.optString("currentBill", "—"),
                balance = d.optString("balance", "—"),
                contractEnd = d.optString("contractEnd", "—"),
                hasUnread = d.optBoolean("hasUnread"),
                ongoingOrders = d.optJSONArray("ongoingOrders").toList(),
                services = d.optJSONArray("services").toList(),
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
            .padding(start = 16.dp, end = 16.dp, top = 24.dp, bottom = 20.dp),
    ) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                Text(data.customerName, color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                Text(" ${data.phoneMasked}", color = Color.White.copy(alpha = 0.85f), fontSize = 13.sp)
            }
            Box {
                Text("✉", color = Color.White, fontSize = 18.sp, modifier = Modifier.clickable { nav.push(Route.Messages) })
                if (data.hasUnread) Box(Modifier.size(8.dp).background(Palette.err, CircleShape).align(Alignment.TopEnd))
            }
        }
        Row(Modifier.padding(top = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            StatusDot()
            Text(data.onlineStatus, color = Color.White, fontSize = 13.sp)
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
            Text(data.planName, fontSize = 16.sp, fontWeight = FontWeight.Bold, color = Palette.ink, modifier = Modifier.weight(1f))
            Tag("在网", Palette.success)
        }
        Row(horizontalArrangement = Arrangement.spacedBy(20.dp)) {
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
        Text(value, fontSize = 14.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
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

private data class QuickEntry(val label: String, val glyph: String, val color: Color, val route: Route)

@Composable
private fun QuickGrid(nav: Nav) {
    val entries = listOf(
        QuickEntry("办套餐", "办", Palette.primary, Route.Products),
        QuickEntry("查订单", "单", Palette.success, Route.Orders),
        QuickEntry("缴费用", "缴", Palette.orange, Route.Pay),
        QuickEntry("报故障", "障", Palette.purple, Route.Fault),
        QuickEntry("充值", "充", Palette.orange, Route.Topup),
        QuickEntry("查用量", "量", Palette.primary, Route.Usage),
        QuickEntry("消息", "信", Palette.success, Route.Messages),
        QuickEntry("客服", "服", Palette.purple, Route.Service),
    )
    Column(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 4.dp)) {
        entries.chunked(4).forEach { row ->
            Row(Modifier.fillMaxWidth()) {
                row.forEach { e ->
                    Column(
                        Modifier.weight(1f).padding(vertical = 6.dp).clickable { nav.push(e.route) },
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Box(Modifier.size(40.dp).background(e.color, RoundedCornerShape(12.dp)), contentAlignment = Alignment.Center) {
                            Text(e.glyph, color = Color.White, fontSize = 18.sp, fontWeight = FontWeight.Bold)
                        }
                        Text(e.label, fontSize = 12.5.sp, color = Palette.ink, modifier = Modifier.padding(top = 6.dp))
                    }
                }
            }
        }
    }
}
