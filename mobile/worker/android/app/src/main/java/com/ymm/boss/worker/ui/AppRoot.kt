package com.ymm.boss.worker.ui

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
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Bg
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary

// 底部 Tab(对齐 nav.js:工作台/工单/我的)
private data class Tab(val screen: Screen, val label: String, val glyph: String)

private val TABS = listOf(
    Tab(Screen.Home, "工作台", "⌂"),
    Tab(Screen.Orders, "工单", "≡"),
    Tab(Screen.Profile, "我的", "◉"),
)

private fun isTabRoot(s: Screen): Boolean = s in TABS.map { it.screen }

@Composable
fun AppRoot(loggedIn: Boolean) {
    val nav = remember { NavHost(if (loggedIn) Screen.Home else Screen.Login) }
    Column(Modifier.fillMaxSize().background(Bg)) {
        Box(Modifier.weight(1f)) {
            when (val cur = nav.current) {
                is Screen.Login -> LoginScreen(onLoggedIn = { nav.reset(Screen.Home) })
                is Screen.Home -> HomeScreen(nav)
                is Screen.Orders -> OrdersScreen(nav)
                is Screen.Profile -> ProfileScreen(nav)
                is Screen.TicketDetail -> TicketDetailScreen(nav, cur.no)
                is Screen.Repair -> PlaceholderScreen(nav, "修复工单", cur.no)
                is Screen.Scan -> PlaceholderScreen(nav, "扫码绑定", cur.no)
                is Screen.Photo -> PlaceholderScreen(nav, "拍照上传", cur.no)
                is Screen.ScanAbnormal -> PlaceholderScreen(nav, "扫码异常", cur.no)
                is Screen.Report -> PlaceholderScreen(nav, "完工上报", cur.no)
                is Screen.Activate -> PlaceholderScreen(nav, "激活", cur.no)
                is Screen.Charge -> PlaceholderScreen(nav, "收费确认", cur.no)
                is Screen.Sign -> PlaceholderScreen(nav, "客户签字", cur.no)
                is Screen.Checkin -> PlaceholderScreen(nav, "到点签到", cur.no)
                is Screen.Navi -> PlaceholderScreen(nav, "一键导航", cur.no)
                is Screen.Transfer -> PlaceholderScreen(nav, "转单", cur.no)
                is Screen.Reschedule -> PlaceholderScreen(nav, "改约", cur.no)
                is Screen.Complaint -> PlaceholderScreen(nav, "报障", cur.no)
                is Screen.Dismantle -> PlaceholderScreen(nav, "拆机", cur.no)
                is Screen.Replace -> PlaceholderScreen(nav, "换机", cur.no)
                is Screen.Retire -> PlaceholderScreen(nav, "退网", cur.no)
                is Screen.Hall -> PlaceholderScreen(nav, "任务池")
                is Screen.Pickup -> PlaceholderScreen(nav, "领料")
                is Screen.Tool -> PlaceholderScreen(nav, "测速工具")
                is Screen.Maintenance -> PlaceholderScreen(nav, "维护清单")
                is Screen.Safety -> PlaceholderScreen(nav, "安全上报")
                is Screen.Schedule -> PlaceholderScreen(nav, "排期")
                is Screen.Performance -> PlaceholderScreen(nav, "绩效明细")
                is Screen.History -> PlaceholderScreen(nav, "历史工单")
                is Screen.Messages -> PlaceholderScreen(nav, "消息中心")
                is Screen.Notice -> PlaceholderScreen(nav, "服务公告")
                is Screen.Help -> PlaceholderScreen(nav, "排障手册")
                is Screen.Service -> PlaceholderScreen(nav, "联系调度")
                is Screen.Settings -> PlaceholderScreen(nav, "接单设置")
                is Screen.Feedback -> PlaceholderScreen(nav, "帮助反馈")
            }
        }
        if (isTabRoot(nav.current)) TabBar(nav)
    }
}

@Composable
private fun TabBar(nav: NavHost) {
    Row(Modifier.fillMaxWidth().background(Color.White).padding(vertical = 6.dp)) {
        TABS.forEach { tab ->
            val active = nav.current == tab.screen
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.weight(1f).clickable { nav.switchTab(tab.screen) }.padding(4.dp),
            ) {
                Text(tab.glyph, fontSize = 18.sp, fontWeight = FontWeight.Bold,
                    color = if (active) Primary else Muted)
                Text(tab.label, fontSize = 11.sp, color = if (active) Primary else Muted)
            }
        }
    }
    Box(Modifier.fillMaxWidth().height(1.dp).background(Line))
}