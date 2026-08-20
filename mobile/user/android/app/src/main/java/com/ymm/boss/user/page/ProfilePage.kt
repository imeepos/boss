package com.ymm.boss.user.page

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
    // 头部固定不滚动;卡片区在上方渐变头之上滚动(后绘者在上),首卡压头 32dp,同首页效果
    var headPx by remember { mutableStateOf(0) }
    val density = LocalDensity.current
    Box(Modifier.fillMaxSize()) {
        ProfileHead(data, nav, Modifier.onSizeChanged { headPx = it.height })
        val topPad = with(density) { headPx.toDp() } - 32.dp
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(top = topPad),
        ) {
            QuickEntriesCard(data, nav)
            ServiceEntriesCard(nav, unread)
            SettingsCard(nav)
            LogoutCard(nav)
            Spacer(Modifier.height(12.dp))
        }
    }
}

@Composable
private fun ProfileHead(data: JSONObject?, nav: Nav, modifier: Modifier = Modifier) {
    val name = data?.optString("name").orEmpty().ifBlank { "加载中…" }
    val verified = data?.optJSONObject("realName")?.optString("status") == "VERIFIED"
    Box(
        modifier
            .fillMaxWidth()
            .background(profileHeaderGradient())
            .statusBarsPadding()
            .padding(start = 16.dp, end = 16.dp, top = 20.dp, bottom = 52.dp),
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
    AppCard(Modifier.fillMaxWidth()) {
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
