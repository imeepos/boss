package com.ymm.boss.user.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.theme.OnGradient

/** 首页/我的 共用的"固定渐变头 + 圆角滚动区"页面骨架,几何参数全在这,保证两页视觉完全一致。 */
object PinnedHeaderSpec {
    /** 固定头部高度(含状态栏区域)。 */
    val headerHeight: Dp = 176.dp

    /** 渐变底尾巴:滚动区顶部圆角缺口透出渐变用。 */
    val gradientTail: Dp = 16.dp

    /** 滚动区与屏幕两侧的边距(与卡片边距一致)。 */
    val sideMargin: Dp = 16.dp

    /** 滚动区顶部圆角。 */
    val corner: Dp = 16.dp
}

/**
 * 结构(自下而上):渐变底(头高+尾巴) → 圆角滚动区(自头部下沿起,不透明底色,超出不可见)
 * → 头部内容层(透明,恒可见)。scrollContent 自带滚动容器(LazyColumn/Column+verticalScroll)。
 */
@Composable
fun PinnedGradientPage(
    gradient: Brush,
    modifier: Modifier = Modifier,
    headerHeight: Dp = PinnedHeaderSpec.headerHeight,
    headerContent: @Composable BoxScope.() -> Unit = {},
    scrollContent: @Composable () -> Unit,
) {
    Box(modifier.fillMaxSize()) {
        Box(
            Modifier
                .fillMaxWidth()
                .height(headerHeight + PinnedHeaderSpec.gradientTail)
                .background(gradient),
        )
        Box(
            Modifier
                .fillMaxSize()
                .padding(top = headerHeight)
                .padding(horizontal = PinnedHeaderSpec.sideMargin)
                .clip(RoundedCornerShape(topStart = PinnedHeaderSpec.corner, topEnd = PinnedHeaderSpec.corner))
                .background(Palette.bg),
        ) { scrollContent() }
        Box(Modifier.fillMaxWidth().height(headerHeight)) { headerContent() }
    }
}

/** 头部右上角消息铃铛(首页用):48dp 触达区,白描边角标可选。 */
@Composable
fun HeaderBell(hasUnread: Boolean, onClick: () -> Unit) {
    Box {
        Icon(
            Icons.Filled.Notifications, contentDescription = "消息通知",
            tint = OnGradient,
            modifier = Modifier
                .size(40.dp)
                .clip(CircleShape)
                .clickable(onClick = onClick)
                .padding(10.dp),
        )
        if (hasUnread) {
            Box(
                Modifier
                    .align(Alignment.TopEnd)
                    .padding(top = 4.dp, end = 6.dp)
                    .size(7.dp)
                    .background(Color(0xFFFF5252), CircleShape),
            )
        }
    }
}

/** 头部小字行(状态/副标题)通用样式。 */
@Composable
fun HeaderStatusDot(text: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(8.dp).background(Color(0xFF34C759), CircleShape))
        Spacer(Modifier.size(6.dp))
        Text(text, fontSize = 14.sp, lineHeight = 16.sp, color = OnGradient, fontWeight = FontWeight.Medium)
    }
}

/** 供 scrollContent 用的列容器快捷参数(卡片默认内边距)。 */
val ScrollCardOuter: PaddingValues = PaddingValues(top = 0.dp, bottom = 6.dp)
