package com.ymm.boss.worker.ui

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Assignment
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.Campaign
import androidx.compose.material.icons.outlined.EventNote
import androidx.compose.material.icons.outlined.HealthAndSafety
import androidx.compose.material.icons.outlined.Inventory
import androidx.compose.material.icons.outlined.MenuBook
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Speed
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.ui.theme.Err
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.Warn
import java.util.Calendar

/** 问候语资源 id(小时段 → 文案),文案见各 values 目录 strings.xml 的 greeting_*。 */
internal fun greetingRes(hour: Int): Int = when (hour) {
    in 5..10 -> R.string.greeting_morning
    in 11..12 -> R.string.greeting_noon
    in 13..17 -> R.string.greeting_afternoon
    else -> R.string.greeting_evening
}

internal data class HomeHeadState(
    val name: String = "",
    val sub: String = "",
    val statusLine: String = "",
    val hasUnread: Boolean = false,
)

/** 渐变头部:问候 + 姓名 / 组别·手机 / 今日状态行 / 消息铃铛(带未读角标)。 */
@Composable
internal fun HomeHeader(state: HomeHeadState, onOpenMessages: () -> Unit) {
    val hour = Calendar.getInstance().get(Calendar.HOUR_OF_DAY)
    Box(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 24.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    "${stringResource(greetingRes(hour))}，${state.name.ifEmpty { stringResource(R.string.default_worker) }}",
                    fontSize = 22.sp, lineHeight = 28.sp, fontWeight = FontWeight.Bold,
                    color = Color.White, maxLines = 1,
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(state.sub, fontSize = 16.sp, lineHeight = 20.sp, color = Color.White.copy(alpha = 0.85f), maxLines = 1)
                Spacer(modifier = Modifier.height(8.dp))
                StatusLine(state.statusLine)
            }
            Box {
                Icon(
                    Icons.Outlined.Notifications, contentDescription = stringResource(R.string.cd_notifications),
                    tint = Color.White, modifier = Modifier
                        .sizeIn(minWidth = 48.dp, minHeight = 48.dp)
                        .clip(CircleShape)
                        .clickable(onClick = onOpenMessages)
                        .padding(12.dp),
                )
                if (state.hasUnread) {
                    Box(
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .padding(top = 8.dp, end = 8.dp)
                            .size(8.dp)
                            .background(Err, CircleShape),
                    )
                }
            }
        }
    }
}

/** 今日概览卡:标题 + 三列数值(对齐 user 端 BroadbandCard 的 SubInfo 布局);顶距 0,贴齐滚动区顶。 */
@Composable
internal fun OverviewCard(today: TodayStats) {
    HomeCard(topPadding = 0) {
        Text(
            stringResource(R.string.home_overview_title), fontSize = 16.sp, lineHeight = 20.sp,
            fontWeight = FontWeight.Bold, color = Primary,
        )
        Row(
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            SubInfo(stringResource(R.string.stat_accepted), today.accepted, Primary, Modifier.weight(1f))
            SubInfo(stringResource(R.string.stat_finished), today.finished, Success, Modifier.weight(1f))
            SubInfo(stringResource(R.string.stat_doing), today.doing, Warn, Modifier.weight(1f))
        }
    }
}

internal data class TodayStats(val accepted: String, val finished: String, val doing: String)

@Composable
private fun SubInfo(label: String, value: String, color: Color, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier.padding(vertical = 6.dp, horizontal = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, fontSize = 13.sp, lineHeight = 16.sp, color = Muted)
        Spacer(modifier = Modifier.height(6.dp))
        Text(value, fontSize = 20.sp, lineHeight = 24.sp, fontWeight = FontWeight.Bold, color = color, maxLines = 1, softWrap = false)
    }
}

/** 白底圆角卡容器(对齐 user 端卡片规格:16dp 圆角 + 16dp 内边距)。 */
@Composable
internal fun HomeCard(modifier: Modifier = Modifier, topPadding: Int = 4, content: @Composable () -> Unit) {
    Card(
        modifier = modifier.fillMaxWidth().padding(top = topPadding.dp, bottom = 4.dp),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = Color.White),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
    ) {
        Column(modifier = Modifier.padding(16.dp)) { content() }
    }
}

/** 卡片头:左侧标题 + 右侧"全部 >"。 */
@Composable
internal fun CardHeader(title: String, more: String? = null, onMore: (() -> Unit)? = null) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold, color = Primary)
        if (more != null) {
            Box(
                modifier = Modifier
                    .sizeIn(minWidth = 48.dp, minHeight = 36.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .clickable(onClick = { onMore?.invoke() }),
                contentAlignment = Alignment.Center,
            ) {
                Text("$more >", fontSize = 13.sp, lineHeight = 16.sp, color = Muted)
            }
        }
    }
}

// 快捷入口(对齐原宫格:排期/公告/任务池/手册/测速/领料/安全/维护);label 存资源 id
private data class QuickAction(@StringRes val label: Int, val icon: ImageVector, val tint: Color, val screen: Screen)

@Composable
internal fun QuickActions(nav: NavHost) {
    val actions = listOf(
        QuickAction(R.string.qa_schedule, Icons.Outlined.EventNote, Primary, Screen.Schedule),
        QuickAction(R.string.qa_notice, Icons.Outlined.Campaign, Success, Screen.Notice),
        QuickAction(R.string.qa_hall, Icons.Outlined.Assignment, Warn, Screen.Hall),
        QuickAction(R.string.qa_survey, Icons.Outlined.LocationOn, Primary, Screen.Surveys),
        QuickAction(R.string.qa_help, Icons.Outlined.MenuBook, Primary, Screen.Help),
        QuickAction(R.string.qa_speed, Icons.Outlined.Speed, Success, Screen.Tool()),
        QuickAction(R.string.qa_pickup, Icons.Outlined.Inventory, Warn, Screen.Pickup),
        QuickAction(R.string.qa_safety, Icons.Outlined.HealthAndSafety, Primary, Screen.Safety),
        QuickAction(R.string.qa_maintenance, Icons.Outlined.Build, Success, Screen.Maintenance),
    )
    HomeCard {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            actions.chunked(4).forEach { rowActions ->
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    rowActions.forEach { a ->
                        GridItem(a, Modifier.weight(1f)) { nav.push(a.screen) }
                    }
                }
            }
        }
    }
}

@Composable
private fun GridItem(action: QuickAction, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .height(64.dp)
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 4.dp, vertical = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(action.icon, contentDescription = stringResource(action.label), tint = action.tint, modifier = Modifier.size(24.dp))
        Text(
            stringResource(action.label), fontSize = 12.sp, lineHeight = 14.sp,
            color = Ink, modifier = Modifier.padding(top = 6.dp),
            textAlign = TextAlign.Center, maxLines = 1,
        )
    }
}

/** 卡内居中提示(加载/空态通用)。 */
@Composable
internal fun CenterHint(text: String) {
    Text(
        text, fontSize = 13.sp, color = Muted,
        modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp),
        textAlign = TextAlign.Center,
    )
}
