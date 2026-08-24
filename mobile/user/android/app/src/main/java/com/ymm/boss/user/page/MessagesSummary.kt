package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Mail
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import kotlinx.coroutines.launch

// 消息中心顶部区域:分类胶囊条(SegmentBar) + 未读摘要卡(SummaryCard) + 标已读失败横幅。

/** 标已读失败横幅:全宽红底白字,3 秒自动消失,不抢占列表错误位。 */
@Composable
internal fun MarkErrBanner(text: String) {
    AppCard(
        outer = PaddingValues(vertical = 6.dp),
        inner = PaddingValues(horizontal = 14.dp, vertical = 10.dp),
    ) {
        Text(text, fontSize = 13.sp, color = Palette.err, modifier = Modifier.fillMaxWidth())
    }
}

@Composable
internal fun SegmentBar(
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
internal fun MessagesSummaryCard(total: Int, unread: Int, loading: Boolean, nav: Nav) {
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
