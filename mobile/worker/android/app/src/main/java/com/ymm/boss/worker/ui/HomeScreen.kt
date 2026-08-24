package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.HomeApi
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import org.json.JSONArray
import org.json.JSONObject

// 工作台首页(结构对齐 user 端首页:固定渐变头 + 圆角滚动区卡片流)
@Composable
fun HomeScreen(nav: NavHost) {
    var reload by remember { mutableIntStateOf(0) }
    val home by loadOnce(reload) { HomeApi.get() }
    val msgs by loadOnce { MiscApi.messages() }

    PinnedGradientPage(
        gradient = Brush.linearGradient(listOf(Primary, Primary2)),
        headerContent = {
            HomeHeader(
                state = homeHead(home, msgs),
                onOpenMessages = { nav.push(Screen.Messages) },
            )
        },
    ) {
        LazyColumn(modifier = Modifier.fillMaxSize()) {
            item {
                when (val s = home) {
                    is Load.Loading -> HomeCard { Loading() }
                    is Load.Fail -> HomeCard { ErrorRetry(stringResource(R.string.err_home_load)) { reload += 1 } }
                    is Load.Ok -> HomeBody(nav, s.data)
                }
            }
            item { QuickActions(nav) }
            item { RemindCard(nav, msgs) }
            item { Spacer(modifier = Modifier.height(16.dp)) }
        }
    }
}

@Composable
private fun HomeBody(nav: NavHost, d: JSONObject) {
    val today = d.optJSONObject("today") ?: JSONObject()
    Column {
        OverviewCard(
            TodayStats(
                accepted = "${today.optInt("accepted")}",
                finished = "${today.optInt("finished")}",
                doing = "${today.optInt("doing")}",
            ),
        )
        val ongoing = d.optJSONArray("ongoing") ?: JSONArray()
        HomeCard(modifier = Modifier.padding(top = 8.dp)) {
            CardHeader(stringResource(R.string.home_ongoing_title, ongoing.length()), more = stringResource(R.string.home_more_all)) { nav.switchTab(Screen.Orders) }
            when {
                ongoing.length() == 0 -> CenterHint(stringResource(R.string.home_no_ongoing))
                else -> for (i in 0 until minOf(ongoing.length(), 3)) {
                    TicketCell(ongoing.optJSONObject(i), onClick = {
                        val no = ongoing.optJSONObject(i).optString("ticketNo")
                        nav.push(ticketScreen(no))
                    })
                }
            }
        }
    }
}

@Composable
private fun RemindCard(nav: NavHost, msgs: Load<JSONObject>) {
    HomeCard(modifier = Modifier.padding(top = 8.dp)) {
        CardHeader(stringResource(R.string.home_reminders), more = stringResource(R.string.home_more_messages), onMore = { nav.push(Screen.Messages) })
        when (msgs) {
            is Load.Loading -> CenterHint(stringResource(R.string.hint_loading))
            is Load.Fail -> CenterHint(stringResource(R.string.err_messages_load))
            is Load.Ok -> {
                val items = msgs.data.optJSONArray("items") ?: JSONArray()
                if (items.length() == 0) CenterHint(stringResource(R.string.home_no_reminders))
                else for (i in 0 until minOf(items.length(), 3)) {
                    val x = items.optJSONObject(i)
                    KvRow(x.optString("title"), x.optString("content"))
                }
            }
        }
    }
}

/** 头部文案:姓名 / 组别·手机号 / 今日状态行 / 未读角标(Message.read 对齐 worker misc 契约)。 */
@Composable
private fun homeHead(home: Load<JSONObject>, msgs: Load<JSONObject>): HomeHeadState = when (home) {
    is Load.Loading -> HomeHeadState(name = stringResource(R.string.default_worker), statusLine = stringResource(R.string.home_status_loading))
    is Load.Fail -> HomeHeadState(name = stringResource(R.string.default_worker), sub = "", statusLine = "")
    is Load.Ok -> {
        val d = home.data
        val today = d.optJSONObject("today") ?: JSONObject()
        HomeHeadState(
            name = d.optString("workerName", stringResource(R.string.default_worker)),
            sub = "${d.optString("groupName")} · ${d.optString("phoneMasked")}",
            statusLine = stringResource(R.string.home_status_fmt, today.optInt("accepted"), today.optInt("doing"), today.optInt("todo")),
            hasUnread = unreadCount(msgs) > 0,
        )
    }
}

private fun unreadCount(msgs: Load<JSONObject>): Int = (msgs as? Load.Ok)
    ?.data?.optJSONArray("items")?.let { a ->
        (0 until a.length()).count { i -> !a.optJSONObject(i).optBoolean("read") }
    } ?: 0

@Composable
fun KvRow(k: String, v: String, valueColor: Color = Ink) {
    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(k, fontSize = 13.sp, color = Muted)
        Text(v, fontSize = 13.sp, color = valueColor, fontWeight = FontWeight.Medium)
    }
}
