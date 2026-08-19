package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.AuthApi
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import kotlinx.coroutines.launch
import org.json.JSONObject

// 我的(对齐 docs/worker/profile.html)
@Composable
fun ProfileScreen(nav: NavHost) {
    val state by loadOnce { ProfileApi.get() }
    val scope = rememberCoroutineScope()

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("个人信息加载失败:${s.message}") }
            is Load.Ok -> ProfileBody(s.data)
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("账户")
            Cell("绩效明细", "提成 · 排名 · 服务评分", onClick = { nav.push(Screen.Performance) })
            Cell("历史工单", onClick = { nav.push(Screen.History) })
            Cell("消息通知", onClick = { nav.push(Screen.Messages) })
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("常用工具")
            Cell("测速 / 光功率", onClick = { nav.push(Screen.Tool()) })
            Cell("领料 / 借还", onClick = { nav.push(Screen.Pickup) })
            Cell("排障手册", onClick = { nav.push(Screen.Help) })
            Cell("联系调度", onClick = { nav.push(Screen.Service) })
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("设置")
            Cell("接单设置", onClick = { nav.push(Screen.Settings) })
            Cell("服务公告", onClick = { nav.push(Screen.Notice) })
            Cell("帮助与反馈", onClick = { nav.push(Screen.Feedback) })
            Text("退出登录", fontSize = 14.sp, color = Color(0xFFCF1322),
                modifier = Modifier.fillMaxWidth()
                    .clickable {
                        scope.launch {
                            try { AuthApi.logout() } catch (_: Exception) { /* 本地照常清理 */ }
                            com.ymm.boss.worker.api.Api.setToken(null)
                            nav.reset(Screen.Login)
                        }
                    }
                    .padding(vertical = 12.dp),
                textAlign = TextAlign.Center)
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun ProfileBody(d: JSONObject) {
    val month = d.optJSONObject("month") ?: JSONObject()
    Column(
        Modifier.fillMaxWidth()
            .background(Brush.linearGradient(listOf(Primary, Primary2)),
                RoundedCornerShape(bottomStart = 22.dp, bottomEnd = 22.dp))
            .padding(horizontal = 16.dp, vertical = 24.dp),
    ) {
        Text(d.optString("name"), color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.Bold)
        Text("${d.optString("groupName")} · 工号 ${d.optString("staffNo")}",
            color = Color.White.copy(alpha = .85f), fontSize = 13.sp)
        Spacer(Modifier.height(14.dp))
        StatusLine("${if (d.optBoolean("online")) "在线接单" else "已停接单"} · 服务 ${d.optInt("serveYears")} 年")
    }
    Box(Modifier.padding(horizontal = 14.dp)) {
        StatCard("本月累计", month.optString("month", ""), listOf(
            Triple("${month.optInt("finished")}", "完成工单", Color(0xFF1677FF)),
            Triple("${month.optInt("onTimeRate")}%", "按时率", Color(0xFF52C41A)),
            Triple("${month.optDouble("score")}", "客户评分", Color(0xFFFAAD14)),
        ))
    }
}
