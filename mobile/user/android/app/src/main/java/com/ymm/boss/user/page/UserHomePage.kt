package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.clickable
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.BottomTabBar
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.brandBlue
import org.json.JSONObject

// ---------- 数据模型:真实 /home 响应(契约 api/openapi/user/schemas.yaml Home) ----------

/** 字段对齐 schemas.yaml Home + docs/contract/fields.md;无 mock,缺省值仅作占位回退。 */
internal data class HomeUiState(
    val loading: Boolean = true,
    val error: String = "",
    val customerName: String = "",
    val phoneMasked: String = "",
    val onlineStatus: String = "",
    val planName: String = "",
    val currentBill: String = "",
    val balance: String = "",
    val contractEnd: String = "",
    val orders: List<HomeOrder> = emptyList(),
    val services: List<HomeService> = emptyList(),
)

internal data class HomeOrder(
    val orderNo: String,
    val status: String,
    val statusLabel: String,
    val productName: String,
    val address: String,
    val stage: Int,
)

internal data class HomeService(
    val name: String,
    val desc: String,
    val statusLabel: String,
)

/** 订单状态中文,枚举对齐 docs/contract/terms.md 第 3 节。 */
internal fun statusLabelOf(status: String): String = when (status) {
    "PENDING" -> "待核查"
    "RESERVED" -> "已预占"
    "INSTALLING" -> "装维中"
    "DONE" -> "已完成"
    "CANCELLED" -> "已取消"
    else -> status
}

internal fun greetingFor(hour: Int): String = when (hour) {
    in 5..10 -> "早上好"
    in 11..12 -> "中午好"
    in 13..17 -> "下午好"
    else -> "晚上好"
}

internal fun parseHome(d: JSONObject): HomeUiState = HomeUiState(
    loading = false,
    customerName = d.optString("customerName"),
    phoneMasked = d.optString("phoneMasked"),
    onlineStatus = d.optString("onlineStatus", "服务在线"),
    planName = d.optJSONObject("plan")?.optString("name").orEmpty(),
    currentBill = yuan(d.optDouble("currentBill", Double.NaN)),
    balance = yuan(d.optDouble("balance", Double.NaN)),
    contractEnd = d.optString("contractEnd"),
    orders = parseOrders(d),
    services = parseServices(d),
)

/** 金额格式:无值返回 "--",有值保留两位小数(契约 L 节 89.00/120.00 形态)。 */
private fun yuan(v: Double): String = if (v.isNaN()) "--" else String.format("¥%.2f", v)

private fun parseOrders(d: JSONObject): List<HomeOrder> {
    val a = d.optJSONArray("ongoingOrders") ?: return emptyList()
    return (0 until a.length()).mapNotNull { i ->
        val o = a.optJSONObject(i) ?: return@mapNotNull null
        val status = o.optString("status")
        HomeOrder(
            orderNo = o.optString("orderNo"),
            status = status,
            statusLabel = o.optString("statusLabel").ifEmpty { statusLabelOf(status) },
            productName = o.optString("productName"),
            address = o.optString("address"),
            stage = o.optInt("stage", 0),
        )
    }
}

private fun parseServices(d: JSONObject): List<HomeService> {
    val a = d.optJSONArray("services") ?: return emptyList()
    return (0 until a.length()).mapNotNull { i ->
        val s = a.optJSONObject(i) ?: return@mapNotNull null
        HomeService(
            name = s.optString("name"),
            desc = s.optString("desc"),
            statusLabel = s.optString("statusLabel", "在网"),
        )
    }
}

// ---------- 页面(结构树对齐规格 D 节) ----------

@Composable
fun UserHomeScreen(nav: Nav) {
    var state by remember { mutableStateOf(HomeUiState()) }
    var reload by remember { mutableIntStateOf(0) }
    LaunchedEffect(reload) {
        state = runCatching { parseHome(UserApi.misc.home()) }
            .getOrElse { e -> state.copy(loading = false, error = "首页加载失败,请重试(${e.message ?: "网络异常"})") }
    }
    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        contentWindowInsets = WindowInsets(0.dp), // 外层 PageScaffold 已做 safeDrawing 避让
        bottomBar = {
            BottomTabBar(nav = nav, currentKey = "home", onSelect = { key -> nav.resetTo(Nav.tabRoute(key)) })
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize()) {
            HomeHeader(state = state, onOpenMessages = { nav.push(Route.Messages) })
            HomeContent(
                state = state,
                padding = padding,
                onRetry = { reload += 1 },
                onAction = { route -> nav.push(route) },
                onOpenOrders = { nav.push(Route.Orders) },
                onOpenOrder = { no -> nav.push(Route.Order(no)) },
                onOpenService = { nav.push(Route.MyPlan) },
            )
        }
    }
}

@Composable
private fun HomeContent(
    state: HomeUiState,
    padding: PaddingValues,
    onRetry: () -> Unit,
    onAction: (Route) -> Unit,
    onOpenOrders: () -> Unit,
    onOpenOrder: (String) -> Unit,
    onOpenService: () -> Unit,
) {
    LazyColumn(modifier = Modifier.fillMaxSize(), contentPadding = padding) {
        item { WelcomeCard(state = state) }
        item { BroadbandCard(state = state, onOpen = onOpenService) }
        item { QuickActions(onAction = onAction) }
        item { OrderSection(state = state, onOpenOrders = onOpenOrders, onOpenOrder = onOpenOrder, onRetry = onRetry) }
        item { MyServiceCard(service = state.services.firstOrNull(), onClick = onOpenService) }
        item { Spacer(modifier = Modifier.height(16.dp)) }
    }
}

/** 加载/错误/空三态都收敛在这一块,不塌布局(验收 6)。 */
@Composable
private fun OrderSection(
    state: HomeUiState,
    onOpenOrders: () -> Unit,
    onOpenOrder: (String) -> Unit,
    onRetry: () -> Unit,
) {
    OrderCardShell(onOpenAll = onOpenOrders) {
        when {
            state.loading -> CenterHint { CircularProgressIndicator(strokeWidth = 2.dp, modifier = Modifier.size(24.dp)) }
            state.error.isNotEmpty() -> ErrorHint(error = state.error, onRetry = onRetry)
            state.orders.isEmpty() -> EmptyHint(text = "暂无进行中订单")
            else -> state.orders.take(2).forEach { o -> OrderItem(order = o, onOpen = onOpenOrder) }
        }
    }
}

@Composable
private fun CenterHint(content: @Composable () -> Unit) {
    Box(modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp), contentAlignment = Alignment.Center) { content() }
}

@Composable
private fun EmptyHint(text: String) {
    Text(
        text, fontSize = 14.sp, color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp), textAlign = TextAlign.Center,
    )
}

@Composable
private fun ErrorHint(error: String, onRetry: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(error, fontSize = 14.sp, color = MaterialTheme.colorScheme.error, textAlign = TextAlign.Center)
        Text(
            "点击重试", fontSize = 14.sp, fontWeight = FontWeight.Medium,
            color = brandBlue(), textAlign = TextAlign.Center,
            modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp).clickable(onClick = onRetry).padding(12.dp),
        )
    }
}
