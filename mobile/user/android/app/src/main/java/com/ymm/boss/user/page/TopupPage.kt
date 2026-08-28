package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.SubmitGuard
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import com.ymm.boss.user.ui.rememberSubmitGuard
import kotlinx.coroutines.launch

/** 对应草稿 docs/user/topup.html:余额充值。GET/POST /topups。 */
@Composable
fun TopupScreen(nav: Nav) {
    var balance by remember { mutableStateOf(0.0) }
    var denoms by remember { mutableStateOf<List<Int>>(emptyList()) }

    LaunchedEffect(nav.refreshTick) {
        try {
            val d = BillApi.balance()
            balance = d.optDouble("balance", 0.0)
            denoms = d.optJSONArray("denominations").optIntList()
        } catch (e: Exception) { /* 保留骨架默认值 */ }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("余额充值", onBack = { nav.pop() }, action = "使用优惠券", onAction = { nav.push(Route.Coupon) })
        TopupFormCard(nav, balance, denoms)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun TopupFormCard(nav: Nav, balance: Double, denoms: List<Int>) {
    var amountText by remember { mutableStateOf("100") }
    var method by remember { mutableStateOf("cash") }
    var payMethods by remember { mutableStateOf(fallbackTopupMethods()) }
    var err by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    // 防重复提交:弱网连点是重复充值创建(重复扣款风险),提交完成前禁用。
    val guard = rememberSubmitGuard()

    LaunchedEffect(nav.refreshTick) {
        // 支付方式后端下发(stripe 可用才含银行卡,未配置默认线下收款);失败回落兜底不阻塞充值。
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
        BalanceHead(balance)
        Notice("充值即时到账,可用于抵扣账单,到期自动结转。")
        DenomRow(denoms) { amountText = it }
        OutlinedTextField(
            value = amountText,
            onValueChange = { amountText = it },
            label = { Text("自定义金额(最低 ¥1)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        FieldLabel("支付方式")
        MethodList(method, payMethods) { method = it }
        if (err.isNotEmpty()) Text(err, fontSize = 12.5.sp, color = Palette.err)
        Button(
            onClick = {
                if (!guard.acquire()) return@Button
                doTopup(scope, nav, amountText, method, guard) { err = it }
            },
            enabled = !guard.active,
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().padding(top = 14.dp).height(44.dp),
        ) { Text(if (guard.active) "充值中…" else "确认充值") }
    }
}

// fallbackTopupMethods 后端未下发时的兜底(同 PayScreen:不含银行卡)。
private fun fallbackTopupMethods(): List<Pair<String, String>> =
    listOf("wechat" to "微信支付", "alipay" to "支付宝", "cash" to "线下收款")

@Composable
private fun BalanceHead(balance: Double) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text("账户余额", fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink,
            modifier = Modifier.weight(1f))
        Tag("¥" + "%.2f".format(balance), Palette.primary)
    }
}

@Composable
private fun DenomRow(denoms: List<Int>, onPick: (String) -> Unit) {
    FieldLabel("充值面额")
    Row(Modifier.fillMaxWidth().padding(bottom = 10.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        denoms.take(3).forEach { n ->
            Text(
                "¥$n",
                fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary,
                modifier = Modifier
                    .weight(1f)
                    .background(Palette.primary.copy(alpha = 0.08f), RoundedCornerShape(8.dp))
                    .clickable { onPick(n.toString()) }
                    .padding(vertical = 10.dp),
            )
        }
    }
}

@Composable
private fun MethodList(current: String, methods: List<Pair<String, String>>, onSelect: (String) -> Unit) {
    // 后端下发支付方式 + 本站特殊项"预付券"(券核销走既有 topup 记账口径)。
    (methods + listOf("prepaid" to "预付券")).forEach { (key, label) ->
        MethodRow(label, current == key) { onSelect(key) }
    }
}

@Composable
private fun MethodRow(label: String, selected: Boolean, onSelect: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable { onSelect() }.padding(vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, fontSize = 14.sp, color = Palette.ink, modifier = Modifier.weight(1f))
        RadioButton(selected = selected, onClick = onSelect)
    }
}

private fun doTopup(
    scope: kotlinx.coroutines.CoroutineScope,
    nav: Nav, amountText: String, method: String,
    guard: SubmitGuard,
    onErr: (String) -> Unit,
) {
    val amount = amountText.toDoubleOrNull()
    if (amount == null || amount < 1.0) { onErr("请输入不低于 ¥1 的充值金额"); return }
    scope.launch {
        try {
            val r = BillApi.topup(amount, method)
            BillApi.lastPayNo = r.optString("payNo")
            nav.push(Route.PayResult)
        } catch (e: Exception) { onErr("充值失败,请重试") } finally { guard.release() }
    }
}

private fun org.json.JSONArray?.optIntList(): List<Int> {
    if (this == null) return emptyList()
    val out = ArrayList<Int>(length())
    for (i in 0 until length()) out.add(optInt(i))
    return out
}

private fun org.json.JSONArray?.optList(): List<org.json.JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<org.json.JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
