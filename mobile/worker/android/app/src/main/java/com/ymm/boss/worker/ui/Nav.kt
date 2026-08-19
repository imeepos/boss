package com.ymm.boss.worker.ui

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

// 页面路由:与 docs/worker/*.html 一一对应,详情类页面携带工单号 no
sealed interface Screen {
    data object Login : Screen
    data object Home : Screen
    data object Orders : Screen
    data object Profile : Screen

    // 工单详情(新装 order.html / 修复 repair.html 按工单前缀分流)
    data class TicketDetail(val no: String) : Screen
    data class Repair(val no: String) : Screen

    // 新装作业链
    data class Scan(val no: String) : Screen          // 扫码绑定
    data class Photo(val no: String) : Screen         // 拍照
    data class ScanAbnormal(val no: String) : Screen  // 扫码异常
    data class Report(val no: String) : Screen        // 结果上报
    data class Activate(val no: String) : Screen      // 激活
    data class Charge(val no: String) : Screen        // 收费
    data class Sign(val no: String) : Screen          // 客户签字

    // 修复/辅助作业
    data class Checkin(val no: String) : Screen       // 到点签到
    data class Navi(val no: String) : Screen          // 一键导航
    data class Transfer(val no: String) : Screen      // 转单
    data class Reschedule(val no: String) : Screen    // 改约
    data class Complaint(val no: String) : Screen     // 报障
    data class Dismantle(val no: String) : Screen     // 拆机
    data class Replace(val no: String) : Screen       // 换机
    data class Retire(val no: String) : Screen        // 退网

    // 独立工具页(快捷入口/菜单可达)
    data object Hall : Screen          // 任务池
    data object Pickup : Screen        // 领料
    data class Tool(val no: String? = null) : Screen  // 测速工具(可携带工单号)
    data object Maintenance : Screen   // 维护清单
    data object Safety : Screen        // 安全上报
    data object Schedule : Screen      // 排期
    data object Performance : Screen   // 绩效
    data object History : Screen       // 历史工单
    data object Messages : Screen      // 消息中心
    data object Notice : Screen        // 公告
    data object Help : Screen          // 手册
    data object Service : Screen       // 联系调度
    data object Settings : Screen      // 接单设置
    data object Feedback : Screen      // 帮助反馈
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

// 工单类型分流:TKT/EMG 前缀走修复工单(repair.html),其余走新装工单(order.html)
fun ticketScreen(no: String): Screen =
    if (no.startsWith("TKT") || no.startsWith("EMG")) Screen.Repair(no) else Screen.TicketDetail(no)