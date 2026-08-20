package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.TabHeader
import org.json.JSONObject

private const val PAGE_SIZE = 10

// 对应设计稿 user-products-orders-profile.png 中屏(账单 tab):状态胶囊 + 订单卡 + 进度步骤条。
// 下拉刷新重拉第 1 页;列表滚动到底部前自动加载下一页(hasMore 由服务端返回)。
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(nav: Nav) {
    var filter by remember { mutableStateOf("all") }
    var orders by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var page by remember { mutableIntStateOf(1) }
    var hasMore by remember { mutableStateOf(true) }
    var loading by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }

    suspend fun load(p: Int) {
        loading = true
        try {
            val resp = OrderApi.list(filter, p, PAGE_SIZE)
            val items = resp.optJSONArray("items").toObjList()
            orders = if (p == 1) items else orders + items
            page = p
            hasMore = resp.optBoolean("hasMore")
            err = ""
        } catch (e: Exception) {
            if (p == 1) err = "订单加载失败,请稍后重试"
        } finally {
            loading = false
        }
    }

    // 下拉刷新统一由 MainActivity 的 PageRefresh 驱动(nav.refreshTick),这里只重拉第 1 页。
    LaunchedEffect(filter, nav.refreshTick) { orders = emptyList(); hasMore = true; load(1) }

    val listState = rememberLazyListState()
    val nearEnd by remember {
        derivedStateOf {
            val info = listState.layoutInfo
            val last = info.visibleItemsInfo.lastOrNull()?.index ?: -1
            info.totalItemsCount > 0 && last >= info.totalItemsCount - 2
        }
    }
    LaunchedEffect(nearEnd, page) {
        if (nearEnd && hasMore && !loading) load(page + 1)
    }

    Column(Modifier.fillMaxSize()) {
        TabHeader("我的订单")
        StatusSeg(filter) { filter = it }
        LazyColumn(state = listState) {
            item { if (err.isNotEmpty()) Notice(err, Palette.err) }
            if (orders.isEmpty() && err.isEmpty() && !loading) item { EmptyState("暂无订单") }
            items(orders) { o -> OrderCard(o, nav) }
            item {
                Text(
                    when {
                        loading && orders.isNotEmpty() -> "加载中..."
                        hasMore -> ""
                        else -> "没有更多订单了"
                    },
                    fontSize = 12.sp, color = Palette.subtle, textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
                )
            }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }
}

@Composable
private fun StatusSeg(current: String, onSelect: (String) -> Unit) {
    androidx.compose.foundation.layout.Row(
        Modifier.padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
    ) {
        listOf("all" to "全部", "in_progress" to "进行中", "done" to "已完成", "cancelled" to "已取消")
            .forEach { (k, label) ->
                PillTab(label, active = k == current, onClick = { onSelect(k) }, plain = true)
            }
    }
}
