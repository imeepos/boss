package com.ymm.boss.worker.ui

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.pulltorefresh.PullToRefreshDefaults
import androidx.compose.material3.pulltorefresh.rememberPullToRefreshState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.api.friendlyMessage
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.TagRed
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 工单列表(样式对齐 user 端账单页):胶囊筛选 tab + 下拉刷新 + 上拉增量渲染
// 文案存资源 id(三语),经 segs() 组合期解析;键名与后端状态过滤口径绑定
private val SEG_KEYS = listOf(
    "doing" to R.string.seg_doing, "todo" to R.string.seg_todo,
    "done" to R.string.seg_done, "all" to R.string.seg_all,
)

@Composable
private fun segs(): List<Pair<String, String>> = SEG_KEYS.map { it.first to stringResource(it.second) }

/** SegRow 默认段位(组合期解析三语)。 */
@Composable
private fun defaultSegs(): List<Pair<String, String>> = segs()

// 非组合上下文取资源(协程 toast 文案)
private fun s(res: Int, vararg fmt: Any = emptyArray()) = Api.context().getString(res, *fmt)

/** 后端 /tickets 一次返回全量(无分页参数),上拉加载按客户端增量渲染。 */
private const val PAGE_SIZE = 20

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(nav: NavHost) {
    var cur by remember { mutableStateOf("doing") }
    var refresh by remember { mutableIntStateOf(0) }
    var tip by remember { mutableStateOf<Pair<Boolean, String>?>(null) } // ok to 文案
    var taking by remember { mutableStateOf("") } // 正在领取的 ticketNo
    var loading by remember { mutableStateOf(true) }
    var refreshing by remember { mutableStateOf(false) }
    var failed by remember { mutableStateOf(false) }
    val items = remember { mutableStateListOf<JSONObject>() }
    var visible by remember { mutableIntStateOf(PAGE_SIZE) }
    val scope = rememberCoroutineScope()
    val pullState = rememberPullToRefreshState()

    // tab 切换/刷新都整表重拉;成功后重置增量窗口
    suspend fun reload() {
        loading = true; failed = false
        try {
            // 进行中 tab 口径与首页 ongoing 一致(TODO+ACCEPTED+DOING 均未完成),拉全量后客户端过滤
            val doingTab = cur == "doing"
            val r = TicketApi.list(if (doingTab || cur == "all") null else cur.uppercase())
            val a = r.optJSONArray("items") ?: JSONArray()
            items.clear()
            for (i in 0 until a.length()) {
                val t = a.optJSONObject(i) ?: continue
                val s = t.optString("status")
                if (doingTab && s != "TODO" && s != "ACCEPTED" && s != "DOING") continue
                items.add(t)
            }
            visible = PAGE_SIZE
        } catch (e: Exception) { failed = true } finally { loading = false }
    }

    LaunchedEffect(cur, refresh) {
        reload()
        // 下拉指示器最短展示 500ms,避免一闪而过
        if (refreshing) { delay(500); refreshing = false }
    }

    fun take(no: String) {
        if (taking.isNotEmpty()) return
        taking = no
        scope.launch {
            tip = try {
                TicketApi.accept(no)
                true to s(R.string.toast_take_ok)
            } catch (e: Exception) {
                false to s(R.string.toast_take_fail, friendlyMessage(e))
            }
            taking = ""
            refresh++
        }
    }

    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = { refreshing = true; refresh++ },
        state = pullState,
        contentAlignment = Alignment.TopStart,
        indicator = {
            // 对齐 user 端 PageRefresh:指示器水平居中,白底容器 + 主色圆环
            PullToRefreshDefaults.Indicator(
                state = pullState,
                isRefreshing = refreshing,
                modifier = Modifier.align(Alignment.TopCenter),
                containerColor = Color.White,
                color = Primary,
            )
        },
    ) {
    Column(Modifier.fillMaxSize()) {
        TopBar(stringResource(R.string.orders_title), action = stringResource(R.string.orders_refresh), onAction = { refresh++ })
        FilterTabs(cur) { cur = it }
        if (tip != null) {
            val (ok, msg) = tip!!
            Text(msg, fontSize = 12.sp, color = if (ok) Success else TagRed.fg,
                modifier = Modifier.padding(horizontal = 16.dp))
        }
        when {
            loading && items.isEmpty() -> HomeCard(topPadding = 0) { CardTitle(stringResource(R.string.orders_card_title)); Loading() }
            failed && items.isEmpty() -> HomeCard(topPadding = 0) { CardTitle(stringResource(R.string.orders_card_title)); Notice(stringResource(R.string.err_orders_load)) }
            else -> TicketList(items.take(visible), hasMore = visible < items.size, nav, onLoadMore = { visible += PAGE_SIZE }, taking = taking) { take(it) }
        }
    }
    }
}

@Composable
private fun TicketList(
    items: List<JSONObject>,
    hasMore: Boolean,
    nav: NavHost,
    onLoadMore: () -> Unit,
    taking: String,
    onTake: (String) -> Unit,
) {
    val listState = rememberLazyListState()
    InfiniteScroll(listState, hasMore, onLoadMore)
    LazyColumn(state = listState, modifier = Modifier.fillMaxSize()) {
        if (items.isEmpty()) {
            item { EmptyState(stringResource(R.string.orders_empty)) }
        }
        items(items, key = { it.optString("ticketNo") }) { t ->
            TicketOrderCard(t, onTake = onTake, taking = taking) { nav.push(ticketScreen(t.optString("ticketNo"))) }
        }
        item {
            Footer(hasMore)
            Spacer(Modifier.height(16.dp))
        }
    }
}

/** 滚到倒数第 3 条触发续载。 */
@Composable
private fun InfiniteScroll(state: LazyListState, hasMore: Boolean, onLoadMore: () -> Unit) {
    LaunchedEffect(state, hasMore) {
        snapshotFlow { state.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1 }
            .collect { last -> if (hasMore && last >= state.layoutInfo.totalItemsCount - 3) onLoadMore() }
    }
}

@Composable
private fun Footer(hasMore: Boolean) {
    Text(
        if (hasMore) stringResource(R.string.orders_load_more) else stringResource(R.string.orders_all_loaded),
        fontSize = 12.sp, color = Muted, textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth().padding(vertical = 10.dp),
    )
}

/** 列表行(对齐 user 端 BillCell:主标题 + 副标题 + 右侧状态列)。 */
/** 胶囊筛选 tab(对齐 user 端 PillTab plain 形态)。 */
@Composable
private fun FilterTabs(current: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        segs().forEach { (key, label) ->
            PillTab(label, active = key == current) { onSelect(key) }
        }
    }
}

@Composable
private fun PillTab(label: String, active: Boolean, onClick: () -> Unit) {
    Text(
        label, fontSize = 13.sp, textAlign = TextAlign.Center,
        color = if (active) Color.White else Muted,
        fontWeight = if (active) FontWeight.W500 else FontWeight.Normal,
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(if (active) Primary else Color(0xFFEEF0F3))
            .border(1.dp, if (active) Primary else Color.Transparent, RoundedCornerShape(999.dp))
            .clickable { onClick() }
            .padding(horizontal = 14.dp, vertical = 7.dp),
    )
}

/** 分段选择器(HistoryScreen 复用,保留原公共签名;默认段位文案三语)。 */
@Composable
fun SegRow(cur: String, onSelect: (String) -> Unit, segs: List<Pair<String, String>> = defaultSegs()) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp)
        .clip(RoundedCornerShape(9.dp))
        .background(Color(0xFFEEF0F3))
        .padding(3.dp)) {
        segs.forEach { (k, label) ->
            val active = k == cur
            Text(label, fontSize = 13.sp,
                color = if (active) Ink else Muted,
                fontWeight = if (active) FontWeight.SemiBold else FontWeight.Normal,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(7.dp))
                    .background(if (active) Color.White else Color.Transparent)
                    .clickable { onSelect(k) }
                    .padding(vertical = 7.dp))
        }
    }
}

/** 空状态:图标 + 文字居中(对齐 user 端 EmptyState)。 */
@Composable
private fun EmptyState(text: String) {
    Column(
        Modifier.fillMaxWidth().padding(vertical = 24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Outlined.Inbox, contentDescription = null, tint = Muted, modifier = Modifier.size(40.dp))
        Spacer(Modifier.height(10.dp))
        Text(text, fontSize = 13.sp, color = Muted)
    }
}
