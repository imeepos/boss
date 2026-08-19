package com.ymm.boss.worker.ui

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

// 页面路由:与 docs/worker/*.html 一一对应,详情页携带工单号 no
sealed interface Screen {
    data object Login : Screen
    data object Home : Screen
    data object Orders : Screen
    data object Profile : Screen
    data class TicketDetail(val no: String) : Screen
}

// 轻量导航栈(底部 Tab 三个根页 + 详情压栈)
class NavHost(initial: Screen) {
    var stack by mutableStateOf(listOf<Screen>(initial))
        private set

    val current: Screen get() = stack.last()

    fun push(s: Screen) {
        stack = stack + s
    }

    fun pop() {
        if (stack.size > 1) stack = stack.dropLast(1)
    }

    // Tab 切换:重置到单根,避免层叠
    fun switchTab(root: Screen) {
        stack = listOf(root)
    }

    fun reset(s: Screen) {
        stack = listOf(s)
    }
}
