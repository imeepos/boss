package com.ymm.boss.user.page

import androidx.compose.foundation.background
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/profile.html:个人中心 tab,聚合 GET /profile。
@Composable
fun ProfileScreen(nav: Nav) {
    var data by remember { mutableStateOf<JSONObject?>(null) }
    LaunchedEffect(Unit) {
        try { data = ProfileApi.get() } catch (e: Exception) { data = null }
    }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        ProfileHead(data)
        RealNameCard(data, nav)
        AddressCard(data, nav)
        PlanCard(data, nav)
        ServiceEntriesCard(nav)
        SettingsCard(nav)
        LanguageCard()
        LogoutCard(nav)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun ProfileHead(data: JSONObject?) {
    val name = data?.optString("name").orEmpty().ifBlank { "加载中…" }
    Column(
        Modifier.fillMaxWidth()
            .background(Brush.linearGradient(listOf(Palette.primary, Palette.primary2)))
            .padding(start = 16.dp, end = 16.dp, top = 24.dp, bottom = 24.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(48.dp).background(Color.White.copy(alpha = 0.25f), CircleShape),
                contentAlignment = Alignment.Center,
            ) { Text(name.take(1), color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.Bold) }
            Column(Modifier.padding(start = 12.dp)) {
                Text(name, color = Color.White, fontSize = 18.sp, fontWeight = FontWeight.Bold)
                Text(data?.optString("phoneMasked").orEmpty(), color = Color.White.copy(alpha = 0.85f), fontSize = 12.5.sp)
            }
        }
    }
}

@Composable
private fun RealNameCard(data: JSONObject?, nav: Nav) {
    val rn = data?.optJSONObject("realName")
    val verified = rn?.optString("status") == "VERIFIED"
    AppCard {
        CardTitle("实名信息", "账号安全") { nav.push(Route.Security) }
        CellRow("姓名", onClick = { nav.push(Route.Verify) }, right = { RightText(rn?.optString("nameMasked") ?: "—") })
        CellRow("证件", onClick = { nav.push(Route.Verify) }) {
            RightText("${rn?.optString("idType").orEmpty()} ${rn?.optString("idNoMasked").orEmpty()}".trim())
        }
        CellRow("核验记录", onClick = { nav.push(Route.Verify) }, right = { Tag(if (verified) "已实名" else "待补登", if (verified) Palette.success else Palette.muted) })
    }
}

@Composable
private fun RightText(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.muted)
}

@Composable
private fun AddressCard(data: JSONObject?, nav: Nav) {
    val list = data?.optJSONArray("addresses").toObjectList()
    AppCard {
        CardTitle("家庭地址", "管理") { nav.push(Route.Address) }
        if (list.isEmpty()) CellRow("暂无地址", desc = "点击管理新增", onClick = { nav.push(Route.Address) })
        list.take(2).forEach { a ->
            CellRow(
                title = a.optString("label"),
                desc = if (a.optBoolean("isDefault")) "默认安装地址" else "备用地址",
                onClick = { nav.push(Route.Address) },
                right = { Tag(if (a.optBoolean("isDefault")) "在用" else "未用", if (a.optBoolean("isDefault")) Palette.success else Palette.muted) },
            )
        }
    }
}

@Composable
private fun PlanCard(data: JSONObject?, nav: Nav) {
    val plan = data?.optJSONObject("plan")
    val fee = plan?.optDouble("monthlyFee", Double.NaN)?.takeIf { !it.isNaN() } ?: 0.0
    AppCard {
        CardTitle("我的套餐", "详情") { nav.push(Route.MyPlan) }
        CellRow(
            title = plan?.optString("name").orEmpty().ifBlank { "—" },
            desc = "¥$fee/月 · 合约至 ${plan?.optString("contractEnd").orEmpty().ifBlank { "—" }}",
            onClick = { nav.push(Route.MyPlan) },
            right = { Tag("在网", Palette.success) },
        )
    }
}

@Composable
private fun ServiceEntriesCard(nav: Nav) {
    val entries = listOf(
        "我的订单" to Route.Orders, "我的账单" to Route.Bills, "缴费记录" to Route.Pay,
        "报障记录" to Route.Fault, "消息中心" to Route.Messages, "优惠券与活动" to Route.Coupon,
        "电子发票" to Route.Invoice,
    )
    AppCard {
        CardTitle("我的服务")
        entries.forEach { (label, route) -> CellRow(label, onClick = { nav.push(route) }) }
    }
}

@Composable
private fun SettingsCard(nav: Nav) {
    val entries = listOf(
        "账号安全" to Route.Security, "通知订阅设置" to Route.Notify, "投诉与建议" to Route.Complaint,
        "帮助中心" to Route.Help, "用户协议与隐私" to Route.Agreement,
    )
    AppCard {
        CardTitle("账号与设置")
        entries.forEach { (label, route) -> CellRow(label, onClick = { nav.push(route) }) }
    }
}

@Composable
private fun LanguageCard() {
    var lang by remember { mutableStateOf("zh") }
    AppCard {
        CardTitle("语言 / Language", null)
        LanguageButtons(lang) { picked ->
            lang = picked
        }
        Text("界面三语由品牌/区域默认语言配置驱动，切换后 3 秒内生效。", fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
    }
}

@Composable
private fun LanguageButtons(lang: String, onPicked: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    Row(Modifier.padding(top = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        listOf("zh" to "中文", "en" to "English", "fil" to "Filipino").forEach { (key, label) ->
            Button(
                onClick = {
                    onPicked(key)
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
}

@Composable
private fun LogoutCard(nav: Nav) {
    val scope = rememberCoroutineScope()
    AppCard {
        Button(
            onClick = {
                scope.launch {
                    try { UserApi.auth.logout() } catch (e: Exception) { } // 端点失败也继续本地登出
                    Api.setToken(null)
                    nav.resetTo(Route.Login)
                }
            },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.err),
            modifier = Modifier.fillMaxWidth().height(44.dp),
        ) { Text("退出登录") }
    }
}
