package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import org.json.JSONArray
import org.json.JSONObject

// 我的 tab(结构对齐 user 端 ProfilePage:渐变头 + 三入口 + 图标瓦片菜单 + 居中红字退出)
@Composable
fun ProfileScreen(nav: NavHost) {
    val profile by loadOnce { ProfileApi.get() }
    val msgs by loadOnce { MiscApi.messages() }

    PinnedGradientPage(
        gradient = Brush.linearGradient(listOf(Primary, Primary2)),
        headerContent = { ProfileHeadContent(profile, nav) },
    ) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
            QuickEntriesCard(quickSubs(profile, msgs), nav)
            ToolsCard(nav)
            SettingsCard(nav, unread = unreadCount(msgs) > 0)
            LogoutCard(nav)
            Spacer(Modifier.height(16.dp))
        }
    }
}

/** 头部:头像(姓名首字) + 姓名 / 组别·工号 / 在线状态行,右侧设置入口。 */
@Composable
private fun ProfileHeadContent(profile: Load<JSONObject>, nav: NavHost) {
    val d = (profile as? Load.Ok)?.data
    Box(
        Modifier.fillMaxSize().padding(start = 16.dp, end = 16.dp, top = 12.dp, bottom = 12.dp),
        contentAlignment = Alignment.CenterStart,
    ) {
        val name = d?.optString("name").orEmpty().ifBlank { "师傅" }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(56.dp)
                    .background(Color.White.copy(alpha = 0.25f), CircleShape)
                    .border(2.dp, Color.White, CircleShape),
                contentAlignment = Alignment.Center,
            ) { Text(name.take(1), color = Color.White, fontSize = 22.sp, fontWeight = FontWeight.Bold) }
            Spacer(Modifier.width(12.dp))
            Column {
                Text(name, color = Color.White, fontSize = 18.sp, fontWeight = FontWeight.Bold)
                Text(
                    "${d?.optString("groupName").orEmpty()} · 工号 ${d?.optString("staffNo").orEmpty()}",
                    color = Color.White.copy(alpha = 0.85f), fontSize = 12.5.sp,
                )
                if (d != null) {
                    StatusLine("${if (d.optBoolean("online")) "在线接单" else "已停接单"} · 服务 ${d.optInt("serveYears")} 年")
                }
            }
        }
        Icon(
            Icons.Filled.Settings, contentDescription = "接单设置", tint = Color.White,
            modifier = Modifier
                .align(Alignment.CenterEnd)
                .size(40.dp)
                .clickable { nav.push(Screen.Settings) }
                .padding(10.dp),
        )
    }
}

/** 三入口副文案:绩效(本月评分) / 历史(完成单数) / 消息(未读数),取不到时给默认占位。 */
private fun quickSubs(profile: Load<JSONObject>, msgs: Load<JSONObject>): QuickSubs {
    val d = (profile as? Load.Ok)?.data
    val month = d?.optJSONObject("month")
    val unread = unreadCount(msgs)
    return QuickSubs(
        performance = month?.let { "评分 ${it.optDouble("score")}" } ?: "提成 · 排名",
        history = month?.let { "本月完成 ${it.optInt("finished")} 单" } ?: "全部工单",
        messages = if (unread > 0) "${unread}条未读" else "无未读",
    )
}

private fun unreadCount(msgs: Load<JSONObject>): Int = (msgs as? Load.Ok)
    ?.data?.optJSONArray("items")?.let { a: JSONArray ->
        (0 until a.length()).count { i -> !a.optJSONObject(i).optBoolean("read") }
    } ?: 0
