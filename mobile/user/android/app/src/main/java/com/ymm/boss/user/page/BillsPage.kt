package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material3.OutlinedButton
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
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/** 对应草稿 docs/user/bills.html:账单列表 + 状态筛选。GET /bills。 */
@Composable
fun BillsScreen(nav: Nav) {
    var due by remember { mutableStateOf(0.0) }
    var period by remember { mutableStateOf("—") }
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var filter by remember { mutableStateOf("") }
    // loading 守卫:加载中不误显「暂无账单」空态。
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(nav.refreshTick) {
        loading = true
        try {
            val d = BillApi.bills()
            due = d.optDouble("currentDue", 0.0)
            period = d.optString("currentPeriod", "—")
            items = d.optJSONArray("items").optList()
        } catch (e: Exception) { /* 骨架保留默认值 */ } finally { loading = false }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("我的账单", onBack = { nav.pop() }, action = "开发票", onAction = { nav.push(Route.Invoice) })
        DueCard(due, period, nav)
        FilterTabs(filter) { filter = it }
        AppCard {
            val shown = when (filter) {
                "paid" -> items.filter { it.optString("status") == "PAID" }
                "unpaid" -> items.filter { it.optString("status") != "PAID" }
                else -> items
            }
            if (shown.isEmpty()) {
                if (loading) Notice("账单加载中…") else EmptyState("暂无账单")
            }
            shown.forEach { b -> BillCell(b) { nav.push(Route.Bill(b.optString("billNo"))) } }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun DueCard(due: Double, period: String, nav: Nav) {
    AppCard(Modifier.fillMaxWidth()) {
        Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            Text("当前应缴(${period} 账期)", fontSize = 12.5.sp, color = Palette.muted)
            Text("¥" + "%.2f".format(due), fontSize = 30.sp, fontWeight = FontWeight.Bold, color = Palette.ink,
                modifier = Modifier.padding(vertical = 6.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { nav.push(Route.Pay) }, colors = ButtonDefaults.buttonColors(containerColor = Palette.primary)) {
                    Text("立即缴费")
                }
                OutlinedButton(onClick = { nav.push(Route.Topup) }) { Text("余额充值", color = Palette.primary) }
            }
        }
    }
}

@Composable
private fun FilterTabs(current: String, onSelect: (String) -> Unit) {
    // 与消息中心/用量页同款 PillTab plain 形态,不再自绘文字 tab
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf("" to "近 6 期", "paid" to "已缴", "unpaid" to "未缴").forEach { (key, label) ->
            PillTab(label, active = key == current, onClick = { onSelect(key) }, plain = true)
        }
    }
}

@Composable
private fun BillCell(b: JSONObject, onClick: () -> Unit) {
    val status = b.optString("status")
    val tagColor = if (status == "PAID") Palette.success else Palette.orange
    CellRow(
        title = "${b.optString("period")} 账期",
        desc = "${b.optString("productName")} · ${b.optString("periodRange")}",
        onClick = onClick,
        right = {
            Column(horizontalAlignment = Alignment.End) {
                Text("¥" + "%.2f".format(b.optDouble("amount")), fontSize = 15.sp,
                    fontWeight = FontWeight.Bold, color = Palette.ink)
                Tag(b.optString("statusLabel"), tagColor)
            }
        },
    )
}

private fun org.json.JSONArray?.optList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
