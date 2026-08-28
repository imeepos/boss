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
import androidx.compose.material3.RadioButton
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
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PrimaryButton
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.SubmitGuard
import com.ymm.boss.user.ui.TopBar
import com.ymm.boss.user.ui.rememberSubmitGuard
import kotlinx.coroutines.launch
import org.json.JSONObject

/** 对应草稿 docs/user/pay.html:在线缴费。POST /payments + GET /payments。 */
@Composable
fun PayScreen(nav: Nav) {
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("在线缴费", onBack = { nav.pop() })
        PayFormCard(nav)
        RecordsCard(nav)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun PayFormCard(nav: Nav) {
    var target by remember { mutableStateOf<JSONObject?>(null) }
    var method by remember { mutableStateOf("cash") }
    var payMethods by remember { mutableStateOf(fallbackPayMethods()) }
    var err by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(nav.refreshTick) {
        target = runCatching { loadUnpaidBill() }.getOrNull()
        if (target == null) err = "账单加载失败"
        // 支付方式后端下发(stripe 可用才含银行卡,未配置默认线下收款);失败回落兜底数组不阻塞缴费。
        runCatching { BillApi.payMethods() }.onSuccess { m ->
            val items = m.optJSONArray("items").optList()
            if (items.isNotEmpty()) {
                payMethods = items.mapNotNull {
                    val key = it.optString("key"); val label = it.optString("label")
                    if (key.isNotBlank()) key to label else null
                }
            }
            if (m.optString("default").isNotBlank()) method = m.optString("default")
        }
    }

    AppCard {
        val amount = target?.optDouble("amount") ?: 0.0
        AmountHead(amount, target)
        Spacer(Modifier.height(8.dp))
        FieldLabel("支付方式")
        MethodGroup(method, payMethods) { method = it }
        if (err.isNotEmpty()) Text(err, fontSize = 12.5.sp, color = Palette.err)
        ConfirmButton(amount, target?.optString("billNo"), method, scope, nav) { err = it }
    }
}

// fallbackPayMethods 后端未下发时的兜底(不含银行卡:card 需 stripe 通道,避免误发起 checkout)。
private fun fallbackPayMethods(): List<Pair<String, String>> =
    listOf("wechat" to "微信支付", "alipay" to "支付宝", "cash" to "线下收款")

private suspend fun loadUnpaidBill(): JSONObject? =
    BillApi.bills().optJSONArray("items").optList().firstOrNull { it.optString("status") == "UNPAID" }

@Composable
private fun AmountHead(amount: Double, target: JSONObject?) {
    val suffix = target?.let { " · ${it.optString("productName")}" } ?: ""
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
        Text("缴费金额", fontSize = 12.5.sp, color = Palette.muted)
        Text("¥" + "%.2f".format(amount), fontSize = 30.sp, fontWeight = FontWeight.Bold,
            color = Palette.primary, modifier = Modifier.padding(vertical = 6.dp))
        Text("账期 ${target?.optString("period") ?: "—"}$suffix", fontSize = 12.sp, color = Palette.muted)
    }
}

@Composable
private fun ConfirmButton(
    amount: Double, billNo: String?, method: String,
    scope: kotlinx.coroutines.CoroutineScope, nav: Nav,
    onErr: (String) -> Unit,
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    // 防重复提交:弱网下连点是重复建支付会话(Stripe 意图 ×2),提交完成前禁用。
    val guard = rememberSubmitGuard()
    PrimaryButton(
        text = if (guard.active) "支付中…" else "确认支付 ¥" + "%.2f".format(amount),
        enabled = billNo != null && amount > 0 && !guard.active,
        modifier = Modifier.fillMaxWidth().padding(top = 14.dp),
        onClick = {
            if (!guard.acquire()) return@PrimaryButton
            doPay(context, scope, nav, billNo, amount, method, onErr, guard)
        },
    )
}

@Composable
private fun MethodGroup(current: String, methods: List<Pair<String, String>>, onSelect: (String) -> Unit) {
    methods.forEach { (key, label) ->
        Row(
            Modifier.fillMaxWidth().clickable { onSelect(key) }.padding(vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(label, fontSize = 14.sp, color = Palette.ink, modifier = Modifier.weight(1f))
            RadioButton(selected = current == key, onClick = { onSelect(key) })
        }
    }
}

private fun doPay(
    context: android.content.Context,
    scope: kotlinx.coroutines.CoroutineScope,
    nav: Nav, billNo: String?, amount: Double, method: String,
    onErr: (String) -> Unit,
    guard: SubmitGuard,
) {
    if (billNo == null) { onErr("暂无待缴账单"); return }
    scope.launch {
        try {
            if (method == "card") {
                // Stripe 托管收银台:后端建会话,系统浏览器跳转支付,落账等 webhook 回调。
                val r = BillApi.stripeCheckout(billNo, amount)
                val url = r.optString("checkoutUrl")
                if (url.isBlank()) { onErr("收银台创建失败,请重试"); return@launch }
                context.startActivity(android.content.Intent(
                    android.content.Intent.ACTION_VIEW, android.net.Uri.parse(url)))
                BillApi.lastPayNo = r.optString("payNo")
            } else {
                val r = BillApi.createPayment(billNo, amount, method)
                BillApi.lastPayNo = r.optString("payNo")
                nav.push(Route.PayResult)
            }
        } catch (e: Exception) { onErr("支付发起失败,请重试") } finally { guard.release() }
    }
}

@Composable
private fun RecordsCard(nav: Nav) {
    var records by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    LaunchedEffect(nav.refreshTick) {
        try {
            records = BillApi.payments().optList()
        } catch (e: Exception) { /* 记录缺省为空 */ }
    }
    AppCard {
        CardTitle("缴费记录", "开发票") { nav.push(Route.Invoice) }
        if (records.isEmpty()) {
            EmptyState("暂无缴费记录")
        }
        records.forEach { r -> RecordCell(r) { nav.push(Route.Receipt(r.optString("payNo"))) } }
    }
}

@Composable
private fun RecordCell(r: JSONObject, onReceipt: () -> Unit) {
    CellRow(
        title = "¥" + "%.2f".format(r.optDouble("amount")) + " · " + r.optString("period") + " 账期",
        desc = BillApi.methodLabel(r.optString("payMethod")) + " · " + r.optString("paidAt"),
        right = {
            Text("凭证", fontSize = 13.sp, color = Palette.primary,
                modifier = Modifier.clickable { onReceipt() })
        },
    )
}

private fun org.json.JSONArray?.optList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
