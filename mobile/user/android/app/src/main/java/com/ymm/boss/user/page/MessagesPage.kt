package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Circle
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.LocalOffer
import androidx.compose.material.icons.outlined.Mail
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Receipt
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应 docs/user/messages.html:消息中心,GET /messages?category= + POST /messages/read-all。
// 设计稿:designs/messages-center-v1.png;规格:designs/messages-center-v1.spec.md。
// 分类样式与 utility 全部抽到 MessageCategory.kt,本文件聚焦页面骨架与交互。
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
        SummaryCard(total = items.size, unread = unread, loading = loading, nav = nav)
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

/** 标已读失败横幅:全宽红底白字,3 秒自动消失,不抢占列表错误位。 */
@Composable
private fun MarkErrBanner(text: String) {
    AppCard(
        outer = PaddingValues(vertical = 6.dp),
        inner = PaddingValues(horizontal = 14.dp, vertical = 10.dp),
    ) {
        Text(text, fontSize = 13.sp, color = Palette.err, modifier = Modifier.fillMaxWidth())
    }
}

/** 列表主体:loading / error / empty / 列表四态。 */
@Composable
private fun MessagesListBody(
    loading: Boolean,
    loadErr: String?,
    items: List<JSONObject>,
    pendingMsgId: String?,
    onRetry: () -> Unit,
    onCardClick: (JSONObject) -> Unit,
) {
    when {
        loading -> LoadingState()
        loadErr != null -> ErrorState(loadErr, onRetry)
        items.isEmpty() -> AppCard { EmptyState("暂无消息") }
        else -> items.forEach { m ->
            MessageCard(
                m = m,
                loading = pendingMsgId == m.optString("messageId"),
                onClick = { onCardClick(m) },
            )
        }
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

@Composable
private fun SegmentBar(
    selected: String,
    totalUnread: Int,
    unreadByCategory: Map<String, Int>,
    onSelect: (String) -> Unit,
) {
    // 5 个分类胶囊在 360dp 屏放不下,加 horizontalScroll 让最后一个 tab 可被滚到。
    Row(
        Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        CATEGORIES.forEach { c ->
            val badge = if (c.key.isEmpty()) {
                if (totalUnread > 0) "$totalUnread" else null
            } else {
                val n = unreadByCategory[c.key] ?: 0
                if (n > 0) "$n" else null
            }
            PillTab(
                label = c.label,
                active = selected == c.key,
                onClick = { onSelect(c.key) },
                icon = c.icon,
                plain = true,
                iconTint = c.iconTint,
                badge = badge,
            )
        }
    }
}

/**
 * 摘要卡:左侧"总数 + 全部消息"图标块,竖向 1dp 分割线,右侧"未读数 + 未读消息"图标块 +
 * 右下角"全部已读"按钮(仅未读>0 时显示)。
 */
@Composable
private fun SummaryCard(total: Int, unread: Int, loading: Boolean, nav: Nav) {
    val scope = rememberCoroutineScope()
    AppCard {
        Row(
            Modifier.fillMaxWidth().height(72.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            StatColumn(loading, total, "全部消息", Icons.Outlined.Mail, Palette.primary, Modifier.weight(1f))
            Box(
                Modifier.height(32.dp).width(1.dp).background(Palette.line),
            )
            StatColumn(loading, unread, "未读消息", Icons.Outlined.Notifications, Palette.primary, Modifier.weight(1f), showAction = unread > 0) {
                scope.launch {
                    try { ProfileApi.readAllMessages() } catch (e: Exception) { }
                    nav.requestRefresh()
                }
            }
        }
    }
}

@Composable
private fun StatColumn(
    loading: Boolean,
    count: Int,
    label: String,
    icon: ImageVector,
    tint: Color,
    modifier: Modifier,
    showAction: Boolean = false,
    onAction: () -> Unit = {},
) {
    Row(modifier, verticalAlignment = Alignment.CenterVertically) {
        Spacer(Modifier.width(8.dp))
        IconTile(icon, tint, size = 32.dp, corner = 10.dp)
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            Text(if (loading) "—" else "$count", fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Palette.ink, maxLines = 1)
            Text(label, fontSize = 11.sp, color = Palette.muted, maxLines = 1)
        }
        if (showAction) {
            Box(
                Modifier
                    .height(40.dp)
                    .widthIn(min = 60.dp)
                    .clickable { onAction() }
                    .padding(horizontal = 4.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    "全部已读", fontSize = 11.sp, color = Palette.primary, fontWeight = FontWeight.W500,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun MessageCard(m: JSONObject, loading: Boolean = false, onClick: () -> Unit) {
    val read = m.optBoolean("read")
    val contentColor = if (read) Palette.subtle else Palette.muted
    val tint = colorOfCategory(m.optString("category"))

    // 不同 category 视觉区分:
    //  - 未读 = 左 3dp × 全卡高 category 色边框 + 卡片底色用 category 色 4% 浅底
    //  - 已读 = 纯白卡,无左边框无底色,只靠 icon + tag 颜色区分类别
    // 警示型(余额/故障)用浅底更突出,信息型(账单/优惠)保持白底,克制不抢眼。
    val cardBg = if (!read) tint.copy(alpha = 0.05f) else Palette.panel

    // IntrinsicSize.Min 让 Row 高度由最高子元素决定,bar 才能 fillMaxHeight() 占满全卡。
    Row(
        Modifier
            .fillMaxWidth()
            .height(IntrinsicSize.Min)
            .padding(horizontal = 14.dp, vertical = 6.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(cardBg)
            .clickable(enabled = !loading) { onClick() },
        verticalAlignment = Alignment.Top,
    ) {
        if (!read) {
            // 左 3dp × 全卡高边框(贴齐卡片左边,圆角由 clip 保证)
            Box(
                Modifier
                    .width(3.dp)
                    .fillMaxHeight()
                    .background(tint),
            )
        }
        Column(
            Modifier
                .weight(1f)
                .padding(14.dp),
        ) {
            MessageCardHeader(m = m, tint = tint, loading = loading)
            if (m.optString("content").isNotBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(
                    m.optString("content"),
                    fontSize = 12.sp, color = contentColor,
                    maxLines = 2, overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

/** 卡片头部:icon + 标题 + tag + 相对时间 + (loading OR 未读点) */
@Composable
private fun MessageCardHeader(m: JSONObject, tint: Color, loading: Boolean) {
    val read = m.optBoolean("read")
    val titleColor = if (read) Palette.muted else Palette.ink
    val titleWeight = if (read) FontWeight.W500 else FontWeight.W600
    val tagLevel = m.optString("tagLevel")
    val tagText = m.optString("tag").ifBlank { categoryLabel(m.optString("category")) }
    val icon = iconOfCategory(m.optString("category"))

    Row(verticalAlignment = Alignment.CenterVertically) {
        IconTile(icon, tint, size = 40.dp, corner = 12.dp)
        Spacer(Modifier.width(12.dp))
        Text(
            m.optString("title").ifBlank { "通知" },
            fontSize = 14.sp, fontWeight = titleWeight, color = titleColor,
            modifier = Modifier.weight(1f), maxLines = 1, overflow = TextOverflow.Ellipsis,
        )
        if (tagText.isNotBlank()) {
            Spacer(Modifier.width(8.dp))
            Tag(tagText, colorOfTag(tagLevel))
        }
        Spacer(Modifier.width(8.dp))
        Text(
            formatTime(m.optString("createdAt")),
            fontSize = 12.sp, color = Palette.muted,
        )
        if (loading) {
            Spacer(Modifier.width(6.dp))
            CircularProgressIndicator(
                strokeWidth = 1.5.dp,
                modifier = Modifier.size(12.dp),
                color = Palette.primary,
            )
        } else if (!read) {
            Spacer(Modifier.width(6.dp))
            Box(Modifier.size(7.dp).background(tint, CircleShape))
        }
    }
}

@Composable
private fun LoadingState() {
    AppCard {
        Box(Modifier.fillMaxWidth().height(120.dp), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(strokeWidth = 2.dp, modifier = Modifier.size(24.dp), color = Palette.primary)
        }
    }
}

@Composable
private fun ErrorState(msg: String, onRetry: () -> Unit) {
    AppCard {
        Column(
            Modifier.fillMaxWidth().clickable { onRetry() }.height(120.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Text(msg, fontSize = 14.sp, color = Palette.err)
            Spacer(Modifier.height(8.dp))
            Text("点击重试", fontSize = 13.sp, color = Palette.primary)
        }
    }
}