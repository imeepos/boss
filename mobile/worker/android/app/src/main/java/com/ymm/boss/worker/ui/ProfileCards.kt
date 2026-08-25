package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.automirrored.filled.Help
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.automirrored.filled.ReceiptLong
import androidx.compose.material.icons.filled.Campaign
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Speed
import androidx.compose.material.icons.filled.Storefront
import androidx.compose.material.icons.filled.TrendingUp
import androidx.compose.material3.Icon
import androidx.compose.material.icons.filled.SystemUpdate
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.api.AuthApi
import com.ymm.boss.worker.api.UpdateApi
import com.ymm.boss.worker.ui.theme.Err
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.Warn
import kotlinx.coroutines.launch

// 我的 tab 卡片区(结构对齐 user 端 ProfileMenu:图标瓦片菜单 + 居中红字退出)

/** 图标瓦片:浅色圆角底 + 彩色图标(对齐 user 端 IconTile 规格)。 */
@Composable
private fun IconTile(icon: ImageVector, tint: Color, size: androidx.compose.ui.unit.Dp = 40.dp) {
    Box(
        modifier = Modifier
            .size(size)
            .background(tint.copy(alpha = 0.12f), RoundedCornerShape(12.dp)),
        contentAlignment = Alignment.Center,
    ) { Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(22.dp)) }
}

@Composable
internal fun MenuRow(icon: ImageVector, tint: Color, label: String, showDot: Boolean = false, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable { onClick() }.padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconTile(icon, tint)
        Spacer(Modifier.width(12.dp))
        Text(label, fontSize = 14.sp, fontWeight = FontWeight.W500, color = Ink, modifier = Modifier.weight(1f))
        if (showDot) {
            Box(Modifier.size(8.dp).background(Err, CircleShape))
            Spacer(Modifier.width(8.dp))
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = Muted, modifier = Modifier.size(20.dp))
    }
}

private data class MenuEntry(val label: String, val icon: ImageVector, val tint: Color, val screen: Screen)

/** 头像下首卡:三入口横排(对齐 user 端 QuickEntriesCard)。 */
@Composable
internal fun QuickEntriesCard(subs: QuickSubs, nav: NavHost) {
    HomeCard(topPadding = 0) {
        Row(Modifier.fillMaxWidth()) {
            QuickEntry(Icons.Filled.TrendingUp, Primary, stringResource(R.string.pc_card_perf), subs.performance, Modifier.weight(1f)) { nav.push(Screen.Performance) }
            QuickEntry(Icons.AutoMirrored.Filled.ReceiptLong, Warn, stringResource(R.string.pc_card_history), subs.history, Modifier.weight(1f)) { nav.push(Screen.History) }
            QuickEntry(Icons.Filled.Notifications, Success, stringResource(R.string.pc_card_msgs), subs.messages, Modifier.weight(1f)) { nav.push(Screen.Messages) }
        }
    }
}

internal data class QuickSubs(val performance: String, val history: String, val messages: String)

@Composable
private fun QuickEntry(icon: ImageVector, tint: Color, label: String, sub: String, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Column(modifier.clickable { onClick() }.padding(vertical = 4.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        IconTile(icon, tint)
        Spacer(Modifier.height(8.dp))
        Text(label, fontSize = 13.sp, fontWeight = FontWeight.W500, color = Ink)
        Spacer(Modifier.height(2.dp))
        Text(sub, fontSize = 11.sp, color = Muted, maxLines = 1)
    }
}

@Composable
internal fun ToolsCard(nav: NavHost) {
    val entries = listOf(
        MenuEntry(stringResource(R.string.pc_card_speed), Icons.Filled.Speed, Primary, Screen.Tool()),
        MenuEntry(stringResource(R.string.pc_card_materials), Icons.Filled.Storefront, Warn, Screen.Pickup),
        MenuEntry(stringResource(R.string.pc_card_troubleshoot), Icons.AutoMirrored.Filled.Help, Success, Screen.Help),
        MenuEntry(stringResource(R.string.pc_card_contact), Icons.AutoMirrored.Filled.Chat, Primary, Screen.Service),
    )
    HomeCard {
        CardTitle(stringResource(R.string.pc_card_tools))
        entries.forEach { e -> MenuRow(e.icon, e.tint, e.label) { nav.push(e.screen) } }
    }
}

@Composable
internal fun SettingsCard(nav: NavHost, unread: Boolean) {
    // 检查更新:手动拉 /client/latest,有新版走 UpdateDialog,无新版提示已是最新。
    val context = androidx.compose.ui.platform.LocalContext.current
    val scope = rememberCoroutineScope()
    var update by remember { mutableStateOf<UpdateApi.UpdateInfo?>(null) }
    var upToDate by remember { mutableStateOf(false) }
    HomeCard {
        CardTitle(stringResource(R.string.pc_card_account))
        MenuRow(Icons.Filled.Settings, Primary, stringResource(R.string.pc_card_jobs)) { nav.push(Screen.Settings) }
        MenuRow(Icons.Filled.Campaign, Warn, stringResource(R.string.pc_card_notices), showDot = unread) { nav.push(Screen.Notice) }
        MenuRow(Icons.AutoMirrored.Filled.Help, Success, stringResource(R.string.pc_card_help)) { nav.push(Screen.Feedback) }
        MenuRow(Icons.Filled.SystemUpdate, Primary, stringResource(R.string.pc_btn_check_update)) {
            upToDate = false
            scope.launch {
                val info = UpdateApi.check(context)
                if (info != null && info.updateAvailable) update = info else upToDate = true
            }
        }
    }
    UpdateDialog(update, onDismiss = { update = null })
    if (upToDate) {
        androidx.compose.material3.AlertDialog(
            onDismissRequest = { upToDate = false },
            title = { Text(stringResource(R.string.pc_up_to_date_title)) },
            text = { Text(stringResource(R.string.pc_version_current, com.ymm.boss.worker.BuildConfig.VERSION_NAME)) },
            confirmButton = {
                androidx.compose.material3.TextButton(onClick = { upToDate = false }) { Text(stringResource(R.string.pc_btn_dismiss_update)) }
            },
        )
    }
}

/** 卡片标题(对齐 user 端 CardTitle)。 */
@Composable
internal fun CardTitle(title: String) {
    Text(title, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Primary, modifier = Modifier.padding(bottom = 4.dp))
}

/** 破坏性操作:白底卡片 + 居中红字(对齐 user 端 LogoutCard)。 */
@Composable
internal fun LogoutCard(nav: NavHost) {
    val scope = rememberCoroutineScope()
    HomeCard {
        Row(
            Modifier.fillMaxWidth().height(40.dp).clickable {
                scope.launch {
                    try { AuthApi.logout() } catch (_: Exception) { } // 端点失败也继续本地登出
                    Api.setToken(null)
                    nav.reset(Screen.Login)
                }
            },
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = null, tint = Err, modifier = Modifier.size(16.dp))
            Spacer(Modifier.width(6.dp))
            Text(stringResource(R.string.pc_btn_logout), fontSize = 15.sp, fontWeight = FontWeight.W500, color = Err)
        }
    }
}
