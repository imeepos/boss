package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ScanApi
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch
import org.json.JSONObject

// 完工上报(对齐 docs/worker/report.html):工单信息 + 备注 + 提交
@Composable
fun ReportScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    var remark by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val toastOk = stringResource(R.string.report_toast_ok)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.report_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.report_load_fail), red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), d.optString("ticketNo"))
                    KvRow(stringResource(R.string.td_customer), "${d.optString("customerName")}(${d.optString("customerPhoneMasked")})")
                    KvRow(stringResource(R.string.scan_kv_address), d.optString("address"))
                }
            }
        }
        if (err.isNotEmpty()) Card(Modifier.padding(12.dp)) { Notice(err, red = true) }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.report_field_remark))
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text(stringResource(R.string.report_remark_hint)) }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
        }
        Card(Modifier.padding(12.dp)) {
            PrimaryButton(stringResource(R.string.report_btn_submit), enabled = !busy, modifier = Modifier.fillMaxWidth()) {
                busy = true
                scope.launch {
                    try {
                        val r = ScanApi.submitReport(no, remark.trim())
                        toast(ctx, r.optString("message", toastOk))
                        nav.pop()
                    } catch (e: Exception) {
                        err = ctx.getString(R.string.report_toast_fail, e.message ?: "")
                        busy = false
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}