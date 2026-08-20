package com.ymm.boss.user.ui

import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.pulltorefresh.PullToRefreshDefaults
import androidx.compose.material3.pulltorefresh.rememberPullToRefreshState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import kotlinx.coroutines.delay

/** 下拉刷新指示器最短展示时长:页面重拉通常由各自 loading 态反馈,这里只保证手势反馈不至于一闪而过。 */
private const val MIN_INDICATOR_MS = 500L

/**
 * 全页面统一下拉刷新容器:在 MainActivity 包裹 RouteScreen,一处配置全局生效。
 * 刷新动作 = nav.refreshTick 自增,各页面用 LaunchedEffect(x, nav.refreshTick) 重拉数据;
 * 指示器配色取主题令牌(surface 容器 + 品牌蓝),深浅色主题随 BossTheme 自动切换。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PageRefresh(nav: Nav, content: @Composable () -> Unit) {
    var refreshing by remember { mutableStateOf(false) }
    val state = rememberPullToRefreshState()
    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = {
            refreshing = true
            nav.requestRefresh()
        },
        state = state,
        modifier = Modifier,
        contentAlignment = Alignment.TopStart,
        indicator = {
            PullToRefreshDefaults.Indicator(
                state = state,
                isRefreshing = refreshing,
                containerColor = MaterialTheme.colorScheme.surface,
                color = brandBlue(),
            )
        },
    ) {
        LaunchedEffect(refreshing) {
            if (refreshing) {
                delay(MIN_INDICATOR_MS)
                refreshing = false
            }
        }
        content()
    }
}
