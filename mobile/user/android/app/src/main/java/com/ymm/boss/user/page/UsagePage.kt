package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 对应草稿 docs/user/usage.html:网络用量,GET /usage?period=,周期切换。
private val PERIODS = listOf(
    "current" to "本周期", "last" to "上月", "six_month" to "近 6 月",
)

@Composable
fun UsageScreen(nav: Nav) {
    var period by remember { mutableStateOf("current") }
    var data by remember { mutableStateOf<JSONObject?>(null) }
    LaunchedEffect(period, nav.refreshTick) {
        try { data = UserApi.misc.usage(period) } catch (e: Exception) { data = null }
    }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("网络用量", onBack = { nav.pop() })
        PeriodBar(period) { period = it }
        SummaryCard(data)
        DetailCard(data)
        OnlineCard(data)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun PeriodBar(selected: String, onSelect: (String) -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 8.dp)) {
        PERIODS.forEach { (key, label) ->
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
private fun SummaryCard(data: JSONObject?) {
    val used = data?.optDouble("used", 0.0) ?: 0.0
    val quota = data?.optDouble("quota", 0.0) ?: 0.0
    val percent = ((data?.optInt("percent", 0)) ?: 0).coerceIn(0, 100)
    val remain = (quota - used).takeIf { it > 0 } ?: 0.0
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 6.dp)
            .background(Palette.panel, RoundedCornerShape(12.dp)).padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(data?.optString("periodLabel").orEmpty().ifBlank { "—" }, fontSize = 12.5.sp, color = Palette.muted)
        Text("${used} ${data?.optString("unit").orEmpty().ifBlank { "GB" }}",
            fontSize = 30.sp, fontWeight = FontWeight.Bold, color = Palette.ink, modifier = Modifier.padding(vertical = 6.dp))
        Text("套餐内含 $quota GB · 剩余 ${"%.1f".format(remain)} GB（${100 - percent}%）", fontSize = 12.sp, color = Palette.muted)
        UsageBar(percent)
        Notice("日均 ${data?.optDouble("dailyAvg", 0.0) ?: 0.0} GB · 预计月底剩余 ${data?.optDouble("forecastRemain", 0.0) ?: 0.0} GB")
    }
}

@Composable
private fun UsageBar(percent: Int) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 12.dp).height(8.dp)
            .background(Palette.subtle.copy(alpha = 0.3f), RoundedCornerShape(4.dp)),
    ) {
        Row(
            Modifier.fillMaxWidth(percent / 100f).background(Palette.primary, RoundedCornerShape(4.dp)),
        ) {}
    }
}

@Composable
private fun DetailCard(data: JSONObject?) {
    val detail = data?.optJSONObject("detail")
    AppCard {
        CardTitle("用量明细")
        CellRow("下行流量", right = { GrayValue("${detail?.optDouble("down", 0.0) ?: 0.0} GB") })
        CellRow("上行流量", right = { GrayValue("${detail?.optDouble("up", 0.0) ?: 0.0} GB") })
        CellRow("IPTV 独立通道", right = { GrayValue(detail?.optString("iptvNote").orEmpty().ifBlank { "—" }) })
    }
}

@Composable
private fun OnlineCard(data: JSONObject?) {
    val online = data?.optJSONObject("online")
    AppCard {
        CardTitle("在线状态")
        CellRow("当前在线时长", right = { GrayValue(online?.optString("duration").orEmpty().ifBlank { "—" }) })
        CellRow("最近一次上线", right = { GrayValue(online?.optString("lastOnlineAt").orEmpty().ifBlank { "—" }) })
    }
}

@Composable
private fun GrayValue(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.muted)
}
