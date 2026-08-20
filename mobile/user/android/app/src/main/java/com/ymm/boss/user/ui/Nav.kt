package com.ymm.boss.user.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.List
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.ShoppingCart
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.List
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/** 极简导航控制器:一个可组合 back stack,避免引入 navigation 依赖。 */
class Nav(initial: Route) {
    private val stack = mutableStateListOf(initial)
    val current: Route get() = stack.last()
    val size: Int get() = stack.size

    /** 全局下拉刷新信号:PageRefresh 触发,各页面作为 LaunchedEffect key 重拉数据。 */
    var refreshTick by mutableIntStateOf(0)
        private set

    fun requestRefresh() { refreshTick++ }

    fun push(route: Route) { stack.add(route) }
    fun replace(route: Route) { stack[stack.size - 1] = route }
    fun pop(): Boolean = if (stack.size > 1) { stack.removeAt(stack.size - 1); true } else false
    fun resetTo(route: Route) { stack.clear(); stack.add(route) }

    companion object {
        /** 底部 tab:与 docs/user/nav.js 一致(首页/服务/账单/我的)。 */
        val TABS = listOf(
            "home" to "首页", "products" to "服务", "orders" to "账单", "profile" to "我的",
        )

        /** tab 图标:选中实心、未选中描边(material-icons-core 随 material3 自带)。 */
        fun tabIcon(key: String, active: Boolean) = when (key) {
            "home" -> if (active) Icons.Filled.Home else Icons.Outlined.Home
            "products" -> if (active) Icons.Filled.ShoppingCart else Icons.Outlined.ShoppingCart
            "orders" -> if (active) Icons.Filled.List else Icons.Outlined.List
            else -> if (active) Icons.Filled.Person else Icons.Outlined.Person
        }

        /** tab key → 路由,PageScaffold 与页面内 Scaffold 底栏共用一份映射。 */
        fun tabRoute(key: String): Route = when (key) {
            "home" -> Route.Home; "products" -> Route.Products
            "orders" -> Route.Orders; else -> Route.Profile
        }
    }
}

@Composable
fun BottomTabBar(nav: Nav, currentKey: String, onSelect: (String) -> Unit) {
    // 页面级 Scaffold 已在外层做 safeDrawing 避让,这里清零默认 insets 防止双重留白。
    // 规格 I/J 节:高度 56dp、无胶囊指示器、选中态主色蓝。
    NavigationBar(
        modifier = Modifier.height(56.dp),
        windowInsets = WindowInsets(0.dp),
        containerColor = MaterialTheme.colorScheme.surface,
        tonalElevation = 0.dp,
    ) {
        Nav.TABS.forEach { (key, label) ->
            val active = key == currentKey
            NavigationBarItem(
                selected = active,
                onClick = { onSelect(key) },
                icon = { Icon(Nav.tabIcon(key, active), contentDescription = label) },
                label = { Text(label, fontSize = 12.sp) },
                colors = NavigationBarItemDefaults.colors(
                    indicatorColor = Color.Transparent,
                    selectedIconColor = brandBlue(),
                    selectedTextColor = brandBlue(),
                    unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
                ),
            )
        }
    }
}

@Composable
fun PageScaffold(
    nav: Nav,
    currentKey: String,
    showTabs: Boolean,
    content: @Composable () -> Unit,
) {
    // edge-to-edge 下必须避让系统栏:否则 TopBar 被状态栏遮挡、底栏被手势条压住。
    // 状态栏区域统一画固定纯色带(与首页一致,不透明、不随页面切换变化)。
    BackHandler(enabled = nav.size > 1) { nav.pop() }
    val base = Modifier.fillMaxSize().imePadding().background(Palette.bg)
        .windowInsetsPadding(WindowInsets.safeDrawing.only(WindowInsetsSides.Bottom + WindowInsetsSides.Horizontal))
    Column(base) {
        StatusBarBand()
        Box(Modifier.weight(1f)) { content() }
        if (showTabs) {
            BottomTabBar(nav, currentKey) { key -> nav.resetTo(Nav.tabRoute(key)) }
        }
    }
}

/** 固定状态栏色带:高度即状态栏 inset,背景色全页面统一为首页渐变起点色。 */
@Composable
private fun StatusBarBand() {
    val height = WindowInsets.statusBars.asPaddingValues().calculateTopPadding()
    Box(Modifier.fillMaxWidth().height(height).background(statusBarSolid()))
}
