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
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Muted

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
