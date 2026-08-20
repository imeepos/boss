package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.automirrored.filled.Help
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.automirrored.filled.ReceiptLong
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.CurrencyYen
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.List
import androidx.compose.material.icons.filled.LocalOffer
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Receipt
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import kotlinx.coroutines.launch

// 我的 tab 菜单区:我的服务/账号与设置/语言/退出登录(设计稿右屏下半部分)。

private data class MenuEntry(val label: String, val icon: ImageVector, val tint: Color, val route: Route)

@Composable
internal fun MenuRow(icon: ImageVector, tint: Color, label: String, showDot: Boolean = false, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable { onClick() }.padding(vertical = 11.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconTile(icon, tint, size = 36.dp)
        Spacer(Modifier.width(12.dp))
        Text(label, fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink, modifier = Modifier.weight(1f))
        if (showDot) {
            Box(Modifier.size(8.dp).background(Palette.err, CircleShape))
            Spacer(Modifier.width(8.dp))
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(20.dp))
    }
}

@Composable
internal fun ServiceEntriesCard(nav: Nav, unread: Int) {
    val entries = listOf(
        MenuEntry("我的订单", Icons.Filled.List, Palette.primary, Route.Orders),
        MenuEntry("我的账单", Icons.AutoMirrored.Filled.ReceiptLong, Palette.primary, Route.Bills),
        MenuEntry("缴费记录", Icons.Filled.CurrencyYen, Palette.orange, Route.Pay),
        MenuEntry("报障记录", Icons.Filled.Build, Palette.purple, Route.Fault),
        MenuEntry("消息中心", Icons.AutoMirrored.Filled.Chat, Palette.purple, Route.Messages),
        MenuEntry("优惠券与活动", Icons.Filled.LocalOffer, Palette.orange, Route.Coupon),
        MenuEntry("电子发票", Icons.Filled.Receipt, Palette.primary, Route.Invoice),
    )
    AppCard {
        CardTitle("我的服务")
        entries.forEach { e ->
            MenuRow(e.icon, e.tint, e.label, showDot = e.route == Route.Messages && unread > 0) {
                nav.push(e.route)
            }
        }
    }
}

@Composable
internal fun SettingsCard(nav: Nav) {
    val entries = listOf(
        MenuEntry("账号安全", Icons.Filled.Lock, Palette.primary, Route.Security),
        MenuEntry("通知订阅设置", Icons.Filled.Notifications, Palette.orange, Route.Notify),
        MenuEntry("投诉与建议", Icons.AutoMirrored.Filled.Chat, Palette.purple, Route.Complaint),
        MenuEntry("帮助中心", Icons.AutoMirrored.Filled.Help, Palette.primary, Route.Help),
        MenuEntry("用户协议与隐私", Icons.Filled.Description, Palette.muted, Route.Agreement),
    )
    AppCard {
        CardTitle("账号与设置")
        entries.forEach { e -> MenuRow(e.icon, e.tint, e.label) { nav.push(e.route) } }
    }
}

@Composable
internal fun LanguageCard() {
    var lang by remember { mutableStateOf("zh") }
    val scope = rememberCoroutineScope()
    AppCard {
        CardTitle("语言 / Language", null)
        Row(Modifier.padding(top = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            listOf("zh" to "中文", "en" to "English", "fil" to "Filipino").forEach { (key, label) ->
                Button(
                    onClick = {
                        lang = key
                        scope.launch {
                            try { ProfileApi.setLanguage(key) } catch (e: Exception) { } // 失败保留本地高亮
                        }
                    },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = if (lang == key) Palette.primary else Palette.panel,
                        contentColor = if (lang == key) Color.White else Palette.muted,
                    ),
                ) { Text(label, fontSize = 12.5.sp) }
            }
        }
        Text(
            "界面三语由品牌/区域默认语言配置驱动,切换后 3 秒内生效。",
            fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp),
        )
    }
}

/** 破坏性操作:白底卡片 + 居中红字,不做实心通栏红按钮(见 designs/FEEDBACK.md)。 */
@Composable
internal fun LogoutCard(nav: Nav) {
    val scope = rememberCoroutineScope()
    AppCard {
        Row(
            Modifier.fillMaxWidth().height(44.dp).clickable {
                scope.launch {
                    try { UserApi.auth.logout() } catch (e: Exception) { } // 端点失败也继续本地登出
                    Api.setToken(null)
                    nav.resetTo(Route.Login)
                }
            },
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = null, tint = Palette.err, modifier = Modifier.size(16.dp))
            Spacer(Modifier.width(6.dp))
            Text("退出登录", fontSize = 15.sp, fontWeight = FontWeight.W500, color = Palette.err)
        }
    }
}
