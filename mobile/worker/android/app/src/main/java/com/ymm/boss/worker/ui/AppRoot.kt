package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.activity.compose.BackHandler
import com.ymm.boss.worker.R
import com.ymm.boss.worker.ui.theme.Bg
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.StatusBarSolid
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.push.DeepLink

// 底部 Tab(对齐 nav.js:工作台/工单/我的);文案走三语资源
private data class Tab(val screen: Screen, val labelRes: Int, val glyph: String)

private val TABS = listOf(
    Tab(Screen.Home, R.string.tab_home, "⌂"),
    Tab(Screen.Orders, R.string.tab_orders, "≡"),
    Tab(Screen.Profile, R.string.tab_profile, "◉"),
)

private fun isTabRoot(s: Screen): Boolean = s in TABS.map { it.screen }

@Composable
fun AppRoot(loggedIn: Boolean) {
    val nav = remember { NavHost(if (loggedIn) Screen.Home else Screen.Login) }
    DisposableEffect(nav) {
        Api.onUnauthorized = {
            Api.setToken(null)
            nav.reset(Screen.Login)
        }
        onDispose {
            if (Api.onUnauthorized != null) Api.onUnauthorized = null
        }
    }
    // 系统返回键:压栈页逐个弹出,栈底则退出
    BackHandler(enabled = nav.stack.size > 1) { nav.pop() }
    // 通知点击深链:未登录时保留 pending,登录后即跳;已在目标页不重复压栈
    LaunchedEffect(DeepLink.pendingNo, loggedIn) {
        val no = DeepLink.pendingNo ?: return@LaunchedEffect
        if (!loggedIn) return@LaunchedEffect
        DeepLink.consume()
        val target = ticketScreen(no)
        if (nav.current != target) nav.push(target)
    }
    Column(
        Modifier.fillMaxSize().background(Bg)
            .windowInsetsPadding(WindowInsets.safeDrawing.only(WindowInsetsSides.Bottom + WindowInsetsSides.Horizontal)),
    ) {
        StatusBarBand()
        Box(Modifier.weight(1f)) {
            when (val cur = nav.current) {
                is Screen.Login -> LoginScreen(
                    onLoggedIn = { nav.reset(Screen.Home) },
                    onOnboard = { nav.push(Screen.Onboard) },
                    onAgreement = { nav.push(Screen.Agreement) })
                is Screen.Onboard -> OnboardScreen(onBack = { nav.pop() }, onAgreement = { nav.push(Screen.Agreement) })
                is Screen.Agreement -> AgreementScreen(nav)
                is Screen.Home -> HomeScreen(nav)
                is Screen.Orders -> OrdersScreen(nav)
                is Screen.Profile -> ProfileScreen(nav)
                is Screen.TicketDetail -> TicketDetailScreen(nav, cur.no)
                // 报障工单(TKT/EMG 前缀)统一走工单详情页,由详情页根据 API 返回自动分支(Repair/Install)
                is Screen.Repair -> RepairScreen(nav, cur.no)
                // 报障工单的修复上报表单(独立页,避免与详情页互相递归)
                is Screen.RepairReport -> RepairReportScreen(nav, cur.no)
                is Screen.Scan -> ScanScreen(nav, cur.no)
                is Screen.Photo -> PhotoScreen(nav, cur.no)
                is Screen.ScanAbnormal -> ScanAbnormalScreen(nav, cur.no)
                is Screen.Report -> ReportScreen(nav, cur.no)
                is Screen.Activate -> ActivateScreen(nav, cur.no)
                is Screen.Charge -> ChargeScreen(nav, cur.no)
                is Screen.Sign -> SignScreen(nav, cur.no)
                is Screen.Checkin -> CheckinScreen(nav, cur.no)
                is Screen.Navi -> NaviScreen(nav, cur.no)
                is Screen.Transfer -> TransferScreen(nav, cur.no)
                is Screen.Reschedule -> RescheduleScreen(nav, cur.no)
                is Screen.Complaint -> ComplaintScreen(nav, cur.no)
                is Screen.Dismantle -> DismantleScreen(nav, cur.no)
                is Screen.Replace -> ReplaceScreen(nav, cur.no)
                is Screen.Retire -> RetireScreen(nav, cur.no)
                is Screen.Hall -> HallScreen(nav)
                is Screen.Pickup -> PickupScreen(nav)
                is Screen.Tool -> ToolScreen(nav, cur.no)
                is Screen.Maintenance -> MaintenanceScreen(nav)
                is Screen.Safety -> SafetyScreen(nav)
                is Screen.Schedule -> ScheduleScreen(nav)
                is Screen.Performance -> PerformanceScreen(nav)
                is Screen.History -> HistoryScreen(nav)
                is Screen.Messages -> MessagesScreen(nav)
                is Screen.Notice -> NoticeScreen(nav)
                is Screen.Help -> HelpScreen(nav)
                is Screen.Service -> ServiceScreen(nav)
                is Screen.Settings -> SettingsScreen(nav)
                is Screen.Feedback -> FeedbackScreen(nav)
            }
        }
        if (isTabRoot(nav.current)) TabBar(nav)
    }
}

/** 固定状态栏色带:高度即状态栏 inset,背景色全页面统一为头部渐变起点色。 */
@Composable
private fun StatusBarBand() {
    val height = WindowInsets.statusBars.asPaddingValues().calculateTopPadding()
    Box(Modifier.fillMaxWidth().height(height).background(StatusBarSolid))
}

@Composable
private fun TabBar(nav: NavHost) {
    Box(Modifier.fillMaxWidth().background(Color.White)) {
        Row(Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
        TABS.forEach { tab ->
            val active = nav.current == tab.screen
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.weight(1f).clickable { nav.switchTab(tab.screen) }.padding(4.dp),
            ) {
                Text(tab.glyph, fontSize = 18.sp, fontWeight = FontWeight.Bold,
                    color = if (active) Primary else Muted)
                Text(stringResource(tab.labelRes), fontSize = 11.sp, color = if (active) Primary else Muted)
            }
        }
        }
        Box(Modifier.fillMaxWidth().height(1.dp).background(Line))
    }
}