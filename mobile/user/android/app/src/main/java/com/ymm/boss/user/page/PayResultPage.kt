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
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/**
 * 对应草稿 docs/user/payresult.html:支付结果页。
 * Route.PayResult 无路由参数,payNo 取支付流程暂存的 BillApi.lastPayNo,
 * 缺省时回退取 GET /payments 最近一条。
 */
@Composable
fun PayResultScreen(nav: Nav) {
    var payNo by remember { mutableStateOf(BillApi.lastPayNo ?: "") }
    var amountLine by remember { mutableStateOf("¥—") }
    var payMethod by remember { mutableStateOf("—") }

    LaunchedEffect(nav.refreshTick) {
        if (payNo.isBlank()) {
            payNo = runCatching { latestPayNo() }.getOrNull() ?: ""
        }
        if (payNo.isNotBlank()) {
            try {
                val r = BillApi.receipt(payNo)
                amountLine = "¥" + "%.2f".format(r.optDouble("amount")) + " · " + r.optString("period") + " 账期"
                payMethod = BillApi.methodLabel(r.optString("payMethod"))
            } catch (e: Exception) { /* 凭证不可达时保留骨架文案 */ }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("支付结果", onBack = { nav.pop() })
        ResultCard(nav, payNo, amountLine, payMethod)
        Notice("若未到账,30 分钟内未到将自动生成工单并人工跟进。")
        Spacer(Modifier.height(12.dp))
    }
}

private suspend fun latestPayNo(): String =
    BillApi.payments().optList().firstOrNull()?.optString("payNo") ?: ""

@Composable
private fun ResultCard(nav: Nav, payNo: String, amountLine: String, payMethod: String) {
    AppCard(Modifier.fillMaxWidth()) {
        Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            Text("✓", fontSize = 44.sp, color = Palette.success)
            Text("支付成功", fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Palette.ink,
                modifier = Modifier.padding(vertical = 4.dp))
            Text(amountLine, fontSize = 13.sp, color = Palette.muted)
        }
        DetailRows(payNo, payMethod)
        Row(Modifier.fillMaxWidth().padding(top = 20.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            OutlinedButton(onClick = { if (payNo.isNotBlank()) nav.push(Route.Receipt(payNo)) },
                modifier = Modifier.weight(1f)) { Text("查看凭证", color = Palette.primary) }
            Button(onClick = { nav.resetTo(Route.Home) },
                colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                modifier = Modifier.weight(1f)) { Text("返回首页") }
        }
    }
}

@Composable
private fun DetailRows(payNo: String, payMethod: String) {
    CellRow(title = "支付方式", right = { Text(payMethod, fontSize = 13.sp, color = Palette.ink) })
    CellRow(title = "交易单号", right = { Text(payNo.ifBlank { "—" }, fontSize = 13.sp, color = Palette.ink) })
    CellRow(title = "到账状态", right = { Tag("已到账 · 复机已生效", Palette.success) })
}

private fun org.json.JSONArray?.optList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
