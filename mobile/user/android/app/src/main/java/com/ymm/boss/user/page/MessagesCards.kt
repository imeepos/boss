package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import org.json.JSONObject

// 消息卡与列表四态(loading / error / empty / 列表)。

/** 列表主体:loading / error / empty / 列表四态。 */
@Composable
internal fun MessagesListBody(
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
