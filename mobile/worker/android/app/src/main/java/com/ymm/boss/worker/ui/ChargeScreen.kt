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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ScanApi
import kotlinx.coroutines.launch

// 现场收款(对齐 docs/worker/charge.html):应收信息 + 实收金额/支付方式 → 转电子签收
@Composable
fun ChargeScreen(nav: NavHost, no: String) {
    val charge by loadOnce(no) { ScanApi.charge(no) }
    var tip by remember { mutableStateOf("") }
    var done by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val defaultMethod = stringResource(R.string.charge_default_method)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.charge_title), onBack = { nav.pop() })
        when (val c = charge) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.charge_load_fail, c.message), red = true) }
            is Load.Ok -> {
                val due = c.data.optDouble("amountDue", 0.0)
                val amountDefault = if (due % 1.0 == 0.0) due.toInt().toString() else due.toString()
                val methods = c.data.optJSONArray("payMethods")
                    ?.let { arr -> List(arr.length()) { arr.optString(it) } }
                    ?.filter { it.isNotBlank() }
                    ?.takeIf { it.isNotEmpty() }
                    ?: listOf("QR", "CASH", "POS")
                var amount by remember(due) { mutableStateOf(amountDefault) }
                var method by remember(methods) { mutableStateOf(methods.firstOrNull() ?: defaultMethod) }
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), c.data.optString("ticketNo", no))
                    val desc = c.data.optString("amountDesc")
                    val dueText = stringResource(R.string.charge_amount_due_fmt, due.toString(), desc)
                    KvRow(stringResource(R.string.charge_amount_due), dueText)
                    KvRow(stringResource(R.string.charge_methods), methods.joinToString(" / "))
                }
                Card(Modifier.padding(12.dp)) {
                    FieldLabel(stringResource(R.string.charge_received_label))
                    OutlinedTextField(value = amount, onValueChange = { amount = it },
                        placeholder = { Text(stringResource(R.string.charge_received_hint)) }, singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        modifier = Modifier.fillMaxWidth())
                    Spacer(Modifier.height(12.dp))
                    FieldLabel(stringResource(R.string.charge_method_label))
                    OptionRow(methods, method) { method = it }
                    Spacer(Modifier.height(12.dp))
                    PrimaryButton(stringResource(R.string.charge_submit), enabled = !done,
                        modifier = Modifier.fillMaxWidth()) {
                        val v = amount.toDoubleOrNull() ?: 0.0
                        if (v <= 0) { tip = ctx.getString(R.string.charge_invalid_amount); return@PrimaryButton }
                        scope.launch {
                            tip = try {
                                val r = ScanApi.submitCharge(no, v, method)
                                done = true // 防误重复提交(需返回 Sign 再进入)
                                val payNo = r.optString("payNo")
                                if (payNo.isNotBlank()) ctx.getString(R.string.charge_done_fmt, payNo)
                                else ctx.getString(R.string.charge_done)
                            } catch (e: Exception) { ctx.getString(R.string.charge_submit_fail, e.message ?: "") }
                        }
                    }
                    if (tip.isNotEmpty()) { Spacer(Modifier.height(8.dp)); Notice(tip, red = !done) }
                }
                Card(Modifier.padding(12.dp)) {
                    Notice(stringResource(R.string.charge_notice))
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}