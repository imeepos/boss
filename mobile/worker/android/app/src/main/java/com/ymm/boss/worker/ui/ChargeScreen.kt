package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.ScanApi
import kotlinx.coroutines.launch

// 现场收款(对齐 docs/worker/charge.html):应收信息 + 实收金额/支付方式 → 转电子签收
@Composable
fun ChargeScreen(nav: NavHost, no: String) {
    val charge by loadOnce(no) { ScanApi.charge(no) }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("现场收款", onBack = { nav.pop() })
        when (val c = charge) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("应收信息加载失败：${c.message}", red = true) }
            is Load.Ok -> {
                val due = c.data.optDouble("amountDue", 0.0)
                val amountDefault = if (due % 1.0 == 0.0) due.toInt().toString() else due.toString()
                val methods = c.data.optJSONArray("payMethods")
                    ?.let { arr -> List(arr.length()) { arr.optString(it) } }
                    ?.filter { it.isNotBlank() }
                    ?.takeIf { it.isNotEmpty() }
                    ?: listOf("QR", "CASH", "POS")
                var amount by remember(due) { mutableStateOf(amountDefault) }
                var method by remember(methods) { mutableStateOf(methods.firstOrNull() ?: "扫码支付") }
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", c.data.optString("ticketNo", no))
                    val desc = c.data.optString("amountDesc")
                    KvRow("应收", "¥$due" + if (desc.isNotEmpty()) "（$desc）" else "")
                    KvRow("收款方式", methods.joinToString(" / "))
                }
                Card(Modifier.padding(12.dp)) {
                    FieldLabel("实收金额")
                    OutlinedTextField(value = amount, onValueChange = { amount = it },
                        placeholder = { Text("¥") }, singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        modifier = Modifier.fillMaxWidth())
                    Spacer(Modifier.height(12.dp))
                    FieldLabel("支付方式")
                    OptionRow(methods, method) { method = it }
                    Spacer(Modifier.height(12.dp))
                    PrimaryButton("确认收款并签收", modifier = Modifier.fillMaxWidth()) {
                        val v = amount.toDoubleOrNull() ?: 0.0
                        if (v <= 0) { tip = "请填写有效的实收金额。"; return@PrimaryButton }
                        scope.launch {
                            tip = try {
                                ScanApi.submitCharge(no, v, method)
                                nav.push(Screen.Sign(no))
                                ""
                            } catch (e: Exception) { "收款提交失败：${e.message}" }
                        }
                    }
                    if (tip.isNotEmpty()) Notice(tip, red = true)
                }
                Card(Modifier.padding(12.dp)) {
                    Notice("收款到账即时更新账务，缴费凭证同步生成电子收据。")
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}