package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CardMembership
import androidx.compose.material.icons.filled.GppGood
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.profileHeaderGradient
import org.json.JSONObject

// 对应设计稿 user-products-orders-profile.png 右屏(我的 tab):渐变头 + 三入口 + 图标菜单。
@Composable
fun ProfileScreen(nav: Nav) {
    var data by remember { mutableStateOf<JSONObject?>(null) }
    var unread by remember { mutableIntStateOf(0) }
    LaunchedEffect(Unit) {
        try { data = ProfileApi.get() } catch (e: Exception) { data = null }
        try {
            unread = UserApi.misc.messages().optJSONArray("items").toObjectList()
                .count { !it.optBoolean("read") }
        } catch (e: Exception) { } // 无红点降级
    }
    // 层次(自下而上):渐变底(=用户信息高+38dp 尾巴) → 滚动卡片区(顶部圆角,首卡
    // 起于用户信息块下沿,压住渐变尾巴形成错位) → 状态栏 scrim → 用户信息层(恒可见)
    var infoPx by remember { mutableStateOf(0) }
    val density = LocalDensity.current
    // 状态栏高度需在 statusBarsPadding 消费之前量取,否则 scrim 拿到 0
    val statusBarDp = WindowInsets.statusBars.asPaddingValues().calculateTopPadding()
    Box(Modifier.fillMaxSize()) {
        // 渐变底:高度 = 用户信息块 + 100px 尾巴,首卡压在尾巴上;交点圆角由首卡顶部圆角呈现
        Box(
            Modifier
                .fillMaxWidth()
                .height(with(density) { (infoPx + HeaderOverlapPx).toDp() })
                .background(profileHeaderGradient()),
        )
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()),
        ) {
            Spacer(Modifier.height(with(density) { infoPx.toDp() }))
            QuickEntriesCard(data, nav)
            ServiceEntriesCard(nav, unread)
            SettingsCard(nav)
            LogoutCard(nav)
            Spacer(Modifier.height(12.dp))
        }
        // 状态栏 scrim:内容滚到顶部时盖住内容,保持渐变底
        Box(
            Modifier
                .fillMaxWidth()
                .height(statusBarDp)
                .background(profileHeaderGradient()),
        )
        // 用户信息层最后绘制:不论怎么滚动,头像/姓名/设置语言始终可见
        ProfileHeadContent(data, nav, Modifier.onSizeChanged { infoPx = it.height })
    }
}

/** 首卡压住渐变尾巴的高度:约 100px,对齐首页 CardOverlap 的错位节奏。 */
private val HeaderOverlapPx = 100

/** 用户信息层:自带渐变底,绘制在 scrim 之上,任何滚动状态可见可读;圆角交给滚动区首卡。 */
@Composable
private fun ProfileHeadContent(data: JSONObject?, nav: Nav, modifier: Modifier = Modifier) {
    val name = data?.optString("name").orEmpty().ifBlank { "加载中…" }
    val verified = data?.optJSONObject("realName")?.optString("status") == "VERIFIED"
    Box(
        modifier
            .fillMaxWidth()
            .background(profileHeaderGradient())
            .statusBarsPadding()
            .padding(start = 16.dp, end = 16.dp, top = 20.dp, bottom = 14.dp),
    ) {
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
                    data?.optString("phoneMasked").orEmpty(),
                    color = Color.White.copy(alpha = 0.85f), fontSize = 12.5.sp,
                )
                if (verified) VerifiedBadge()
            }
        }
        SettingsAndLanguage(Modifier.align(Alignment.TopEnd), nav)
    }
}

/** 顶部右上角:语言下拉 + 设置入口,同一行垂直居中。 */
@Composable
private fun SettingsAndLanguage(modifier: Modifier = Modifier, nav: Nav) {
    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        LanguageDropdown()
        Spacer(Modifier.width(8.dp))
        Icon(
            Icons.Filled.Settings, contentDescription = "设置", tint = Color.White,
            modifier = Modifier.size(40.dp).clickable { nav.push(Route.Security) }.padding(10.dp),
        )
    }
}

@Composable
private fun VerifiedBadge() {
    Row(
        Modifier.padding(top = 6.dp)
            .background(Color.White.copy(alpha = 0.2f), RoundedCornerShape(999.dp))
            .padding(horizontal = 8.dp, vertical = 3.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(Icons.Filled.GppGood, contentDescription = null, tint = Color.White, modifier = Modifier.size(12.dp))
        Spacer(Modifier.width(4.dp))
        Text("已实名", color = Color.White, fontSize = 11.sp)
    }
}

@Composable
private fun QuickEntriesCard(data: JSONObject?, nav: Nav) {
    val verified = data?.optJSONObject("realName")?.optString("status") == "VERIFIED"
    val addrCount = data?.optJSONArray("addresses")?.length() ?: 0
    val planName = data?.optJSONObject("plan")?.optString("name").orEmpty().ifBlank { "—" }
    AppCard(
        Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp, bottomStart = 12.dp, bottomEnd = 12.dp),
    ) {
            Row(Modifier.fillMaxWidth()) {
                QuickEntry(Icons.Filled.GppGood, Palette.primary, "实名信息", if (verified) "已实名" else "待补登", Modifier.weight(1f)) {
                    nav.push(Route.Verify)
                }
                QuickEntry(Icons.Filled.Home, Palette.orange, "家庭地址", "${addrCount}个地址", Modifier.weight(1f)) {
                    nav.push(Route.Address)
                }
                QuickEntry(Icons.Filled.CardMembership, Palette.purple, "我的套餐", planName, Modifier.weight(1f)) {
                    nav.push(Route.MyPlan)
                }
            }
        }
}

@Composable
private fun QuickEntry(
    icon: ImageVector, tint: Color, label: String, sub: String,
    modifier: Modifier = Modifier, onClick: () -> Unit,
) {
    Column(modifier.clickable { onClick() }.padding(vertical = 4.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        IconTile(icon, tint, size = 40.dp, corner = 12.dp)
        Spacer(Modifier.height(8.dp))
        Text(label, fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink)
        Spacer(Modifier.height(2.dp))
        Text(sub, fontSize = 11.sp, color = Palette.muted, maxLines = 1)
    }
}
