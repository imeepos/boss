package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/messages.html:消息中心,GET /messages?category= + POST /messages/read-all。
private val CATEGORIES = listOf(
    "" to "全部", "billing" to "账单缴费", "balance" to "余额预警", "fault" to "故障公告", "promo" to "优惠活动",
)

@Composable
fun MessagesScreen(nav: Nav) {
    var category by remember { mutableStateOf("") }
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    LaunchedEffect(category) {
        try {
            val d = UserApi.misc.messages(category.ifBlank { null })
            items = d.optJSONArray("items").toObjectList()
        } catch (e: Exception) { items = emptyList() }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("消息中心", { nav.pop() }, "订阅设置") { nav.push(Route.Notify) }
        SegmentBar(category) { category = it }
        AppCard {
            if (items.isEmpty()) Text("暂无消息", fontSize = 12.5.sp, color = Palette.muted)
            items.forEach { m -> MessageCell(m, nav) }
        }
        ReadAllButton { category = "" }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun ReadAllButton(onDone: () -> Unit) {
    val scope = rememberCoroutineScope()
    AppCard {
        Button(
            onClick = {
                scope.launch {
                    try { ProfileApi.readAllMessages() } catch (e: Exception) { } // 已读状态由列表刷新体现
                    onDone()
                }
            },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().height(42.dp),
        ) { Text("全部标为已读") }
    }
}

@Composable
private fun SegmentBar(selected: String, onSelect: (String) -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 8.dp)) {
        CATEGORIES.forEach { (key, label) ->
            Text(
                label,
                fontSize = 13.sp,
                fontWeight = if (selected == key) FontWeight.Bold else FontWeight.Normal,
                color = if (selected == key) Palette.primary else Palette.muted,
                modifier = Modifier.padding(end = 16.dp).clickable { onSelect(key) },
            )
        }
    }
}

@Composable
private fun MessageCell(m: JSONObject, nav: Nav) {
    CellRow(
        title = m.optString("title"),
        desc = m.optString("content"),
        onClick = { nav.push(routeOf(m.optString("category"))) },
        right = { Tag(m.optString("tag").ifBlank { "通知" }, colorOf(m.optString("tagLevel"))) },
    )
}

private fun routeOf(category: String): Route = when (category) {
    "balance" -> Route.Topup
    "fault" -> Route.Fault
    "promo" -> Route.Coupon
    else -> Route.Bills
}

private fun colorOf(tagLevel: String): Color = when (tagLevel) {
    "balance", "bill" -> Palette.orange
    "promo" -> Palette.err
    "info" -> Palette.primary
    else -> Palette.muted
}
