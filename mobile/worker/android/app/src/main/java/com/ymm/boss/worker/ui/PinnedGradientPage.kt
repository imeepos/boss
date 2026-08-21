package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.ui.theme.Bg

/**
 * 对齐 user 端 PinnedGradientPage 的"固定渐变头 + 圆角滚动区"骨架,
 * 几何参数一致,底色换成 worker 的 Bg,供工作台首页使用。
 */
object PinnedHeaderSpec {
    val headerHeight: Dp = 176.dp
    val gradientTail: Dp = 16.dp
    val sideMargin: Dp = 16.dp
    val corner: Dp = 16.dp
    val regionLift: Dp = 46.dp
}

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
                .padding(top = headerHeight - PinnedHeaderSpec.regionLift)
                .padding(horizontal = PinnedHeaderSpec.sideMargin)
                .clip(RoundedCornerShape(topStart = PinnedHeaderSpec.corner, topEnd = PinnedHeaderSpec.corner))
                .background(Bg),
        ) { scrollContent() }
        Box(Modifier.fillMaxWidth().height(headerHeight)) { headerContent() }
    }
}
