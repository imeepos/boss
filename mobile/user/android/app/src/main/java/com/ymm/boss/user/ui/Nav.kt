package com.ymm.boss.user.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/** 极简导航控制器:一个可组合 back stack,避免引入 navigation 依赖。 */
class Nav(initial: Route) {
    private val stack = mutableStateListOf(initial)
    val current: Route get() = stack.last()
    val size: Int get() = stack.size

    fun push(route: Route) { stack.add(route) }
    fun replace(route: Route) { stack[stack.size - 1] = route }
    fun pop(): Boolean = if (stack.size > 1) { stack.removeAt(stack.size - 1); true } else false
    fun resetTo(route: Route) { stack.clear(); stack.add(route) }

    companion object {
        /** 底部 tab:与 docs/user/nav.js 一致(首页/产品/订单/我的)。 */
        val TABS = listOf(
            "home" to "首页", "products" to "产品", "orders" to "订单", "profile" to "我的",
        )
    }
}

@Composable
fun BottomTabBar(nav: Nav, currentKey: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.fillMaxWidth().height(56.dp).background(Palette.panel),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Nav.TABS.forEach { (key, label) ->
            val active = key == currentKey
            Column(
                Modifier.weight(1f).fillMaxSize().clickable { onSelect(key) },
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center,
            ) {
                Text(if (active) "●" else "○", color = if (active) Palette.primary else Palette.subtle, fontSize = 14.sp)
                Text(label, color = if (active) Palette.primary else Palette.muted, fontSize = 12.sp, fontWeight = if (active) FontWeight.W600 else FontWeight.Normal)
            }
        }
    }
}

@Composable
fun PageScaffold(nav: Nav, currentKey: String, showTabs: Boolean, content: @Composable () -> Unit) {
    Column(Modifier.fillMaxSize().background(Palette.bg)) {
        Box(Modifier.weight(1f)) { content() }
        if (showTabs) {
            BottomTabBar(nav, currentKey) { key ->
                val target = when (key) {
                    "home" -> Route.Home; "products" -> Route.Products
                    "orders" -> Route.Orders; else -> Route.Profile
                }
                nav.resetTo(target)
            }
        }
    }
}
