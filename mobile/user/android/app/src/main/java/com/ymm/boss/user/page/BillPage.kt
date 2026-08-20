package com.ymm.boss.user.page

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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.BillApi
import kotlinx.coroutines.launch
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/** 对应草稿 docs/user/bill.html:账单明细。GET /bills/{billNo}。 */
@Composable
fun BillScreen(nav: Nav, no: String) {
    var bill by remember { mutableStateOf<JSONObject?>(null) }
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var total by remember { mutableStateOf(0.0) }
    var autoPay by remember { mutableStateOf(false) }
    var failed by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(no, nav.refreshTick) {
        try {
            val d = BillApi.billDetail(no)
            bill = d.optJSONObject("bill")
            items = d.optJSONArray("items").optList()
            total = d.optDouble("totalDue", 0.0)
            autoPay = BillApi.autoPay().optBoolean("autoPayEnabled", d.optBoolean("autoPayEnabled"))
        } catch (e: Exception) { failed = true }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("账单明细", onBack = { nav.pop() }, action = "开发票", onAction = { nav.push(Route.Invoice) })
        DetailCard(bill, items, total, failed)
        AutoPayCard(autoPay) { enabled -> scope.launch { toggleAutoPay(enabled) { autoPay = it } } }
        Button(
            onClick = { nav.push(Route.Pay) },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp).height(44.dp),
        ) { Text("立即缴费 ¥" + "%.2f".format(total)) }
        Spacer(Modifier.height(8.dp))
    }
}

@Composable
private fun DetailCard(bill: JSONObject?, items: List<JSONObject>, total: Double, failed: Boolean) {
    AppCard {
        CardTitle(if (bill != null) "${bill.optString("period")} 账期" else "加载中…")
        if (bill != null) BillHeader(bill)
        if (failed) Notice("账单加载失败,请稍后重试", Palette.err)
        items.forEach { FeeCell(it) }
        Row(Modifier.fillMaxWidth().padding(top = 12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("合计应缴", fontSize = 14.sp, fontWeight = FontWeight.Bold, color = Palette.ink,
                modifier = Modifier.weight(1f))
            Text("¥" + "%.2f".format(total), fontSize = 16.sp, fontWeight = FontWeight.Bold,
                color = Palette.primary)
        }
    }
}

@Composable
private fun BillHeader(b: JSONObject) {
    Row(Modifier.padding(top = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        Tag(b.optString("statusLabel"), if (b.optString("status") == "PAID") Palette.success else Palette.orange)
        Text(" ${b.optString("productName")} · ${b.optString("periodRange")}",
            fontSize = 12.sp, color = Palette.muted)
    }
}

@Composable
private fun AutoPayCard(autoPay: Boolean, onToggle: (Boolean) -> Unit) {
    AppCard {
        CardTitle("自动缴费")
        Tag(if (autoPay) "已开通" else "未开通", if (autoPay) Palette.success else Palette.muted)
        Notice("开通后每月账期自动扣款,避免忘记缴费导致停机。")
        OutlinedButton(onClick = { onToggle(!autoPay) }) {
            Text(if (autoPay) "关闭自动缴费" else "开通微信自动缴费", color = Palette.primary)
        }
    }
}

// POST billing/auto-pay 切换开通状态;失败保持原状态并提示。
private suspend fun toggleAutoPay(enabled: Boolean, apply: (Boolean) -> Unit) {
    try {
        apply(BillApi.setAutoPay(enabled).optBoolean("autoPayEnabled", enabled))
    } catch (e: Exception) { apply(!enabled) }
}

@Composable
private fun FeeCell(item: JSONObject) {
    val amount = item.optDouble("amount")
    val positive = amount >= 0
    CellRow(
        title = item.optString("name"),
        desc = item.optString("range"),
        right = {
            Text(
                if (positive) "¥" + "%.2f".format(amount) else "-¥" + "%.2f".format(-amount),
                fontSize = 14.sp, fontWeight = FontWeight.W500,
                color = if (positive) Palette.ink else Palette.success,
            )
        },
    )
}

private fun org.json.JSONArray?.optList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
