package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应 docs/user/messages.html:消息中心,GET /messages?category= + POST /messages/read-all。
// 设计稿:designs/messages-center-v1.png;规格:designs/messages-center-v1.spec.md。
// 分类样式与 utility 在 MessageCategory.kt;摘要卡/胶囊条在 MessagesSummary.kt;消息卡在 MessagesCards.kt。
@Composable
fun MessagesScreen(nav: Nav) {
    var category by remember { mutableStateOf("") }
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var loadErr by remember { mutableStateOf<String?>(null) }
    // pendingMsgId:正在调用单条已读接口的 messageId,用于卡片显示 loading 与去重点击。
    var pendingMsgId by remember { mutableStateOf<String?>(null) }
    // markErr:点击标已读失败的瞬时错误(顶部红条 3 秒自动消失),不抢占列表错误位。
    var markErr by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    // markErr 自动清除:出现后 3s 淡出,避免阻塞后续操作。
    LaunchedEffect(markErr) {
        if (markErr != null) {
            kotlinx.coroutines.delay(3000)
            markErr = null
        }
    }

    LaunchedEffect(category, nav.refreshTick) {
        loading = true
        loadErr = null
        try {
            val d = UserApi.misc.messages(category.ifBlank { null })
            items = d.optJSONArray("items").toObjectList()
        } catch (e: Exception) {
            loadErr = e.message ?: "加载失败"
            items = emptyList()
        } finally {
            loading = false
        }
    }

    // 全量统计未读数:每次刷新拉一次,按 category 分组。"全部" tab 用 totalUnread,其他 tab 用各自分组未读。
    // 失败静默 → 空 map → tab 不显示徽章,不打扰用户。
    var unreadByCategory by remember { mutableStateOf<Map<String, Int>>(emptyMap()) }
    var totalUnread by remember { mutableIntStateOf(0) }
    LaunchedEffect(nav.refreshTick) {
        try {
            val d = UserApi.misc.messages(null)
            val all = d.optJSONArray("items").toObjectList()
            totalUnread = all.count { !it.optBoolean("read") }
            unreadByCategory = all.groupBy { it.optString("category") }
                .mapValues { (_, list) -> list.count { !it.optBoolean("read") } }
        } catch (e: Exception) { }
    }

    val unread = items.count { !it.optBoolean("read") }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("消息中心", onBack = { nav.pop() }, action = "订阅设置", onAction = { nav.push(Route.Notify) })
        SegmentBar(category, totalUnread, unreadByCategory) { category = it }
        MessagesSummaryCard(total = items.size, unread = unread, loading = loading, nav = nav)
        if (markErr != null) MarkErrBanner(markErr!!)
        MessagesListBody(
            loading = loading,
            loadErr = loadErr,
            items = items,
            pendingMsgId = pendingMsgId,
            onRetry = { category = category },
            onCardClick = { m ->
                nav.handleMessageTap(
                    m = m,
                    scope = scope,
                    onPendingChange = { pendingMsgId = it },
                    onMarkErr = { markErr = it },
                )
            },
        )
        Spacer(Modifier.height(12.dp))
    }
}

/**
 * 点击单条消息的行为:
 *  - 已读 → 直接跳转详情
 *  - 未读 → 先调 PUT /messages/{id}/read,成功后再跳转;失败显示顶部 markErr 横幅
 */
private fun Nav.handleMessageTap(
    m: JSONObject,
    scope: kotlinx.coroutines.CoroutineScope,
    onPendingChange: (String?) -> Unit,
    onMarkErr: (String) -> Unit,
) {
    val msgId = m.optString("messageId")
    val wasRead = m.optBoolean("read")
    if (wasRead) {
        push(routeOf(m.optString("category")))
        return
    }
    onPendingChange(msgId)
    scope.launch {
        val ok = try {
            ProfileApi.readMessage(msgId)
            true
        } catch (e: Exception) {
            android.util.Log.w("MessagesPage",
                "mark-read failed msgId=$msgId: ${e::class.simpleName} ${e.message}", e)
            onMarkErr("标记已读失败,请重试")
            false
        }
        onPendingChange(null)
        if (ok) push(routeOf(m.optString("category")))
    }
}
