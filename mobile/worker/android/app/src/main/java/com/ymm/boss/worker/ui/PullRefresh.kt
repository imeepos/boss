package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Box
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
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.delay

/** 指示器最短展示时长:保证手势反馈不至于一闪而过(后端响应通常几十 ms)。 */
private const val MIN_INDICATOR_MS = 500L

/**
 * 页面级下拉刷新容器:
 * - onRefresh 回调自增 nav.refreshTick,触发本页面 loadOnce 重拉
 * - 指示器居中顶部,配色取主蓝 Primary(#086CF5)
 *
 * 注意:容器高度由父布局决定;若内容是 Column 而非 LazyColumn/verticalScroll,
 * PullToRefreshBox 会按子项 intrinsic 高度给到手势空间,这里默认无 Modifier 即可。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PageRefresh(nav: NavHost, content: @Composable () -> Unit) {
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
                modifier = Modifier.align(Alignment.TopCenter),
                containerColor = MaterialTheme.colorScheme.surface,
                color = Primary,
            )
        },
    ) {
        Box(modifier = Modifier) { content() }
    }
    LaunchedEffect(refreshing) {
        if (refreshing) {
            delay(MIN_INDICATOR_MS)
            refreshing = false
        }
    }
}