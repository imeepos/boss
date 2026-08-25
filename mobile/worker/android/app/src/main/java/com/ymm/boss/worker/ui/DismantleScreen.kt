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
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 拆机作业(对齐 docs/worker/dismantle.html):扫码解绑释放端口
@Composable
fun DismantleScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val unboundOk = stringResource(R.string.dismantle_unbound_ok)
    val unboundFail = stringResource(R.string.dismantle_unbound_fail)
    val errBlank = stringResource(R.string.dismantle_err_blank)
    var epc by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.dismantle_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.dismantle_load_fail), red = true) }
            is Load.Ok -> Card(Modifier.padding(12.dp)) {
                KvRow(stringResource(R.string.td_ticket_no), s.data.optString("ticketNo"))
                KvRow(stringResource(R.string.dismantle_kv_prebind), s.data.optString("preBindTag", "-"))
                KvRow(stringResource(R.string.dismantle_kv_splitter), s.data.optString("splitterPort", "-"))
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.dismantle_field_epc))
            OutlinedTextField(value = epc, onValueChange = { epc = it },
                placeholder = { Text(stringResource(R.string.dismantle_epc_hint)) }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton(stringResource(R.string.dismantle_submit), modifier = Modifier.fillMaxWidth()) {
                if (epc.isBlank()) { tip = errBlank; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = AssetApi.dismantleScan(no, epc.trim())
                        val msg = if (r.optBoolean("unbound") && r.optBoolean("portReleased"))
                            unboundOk else r.optString("message", unboundFail)
                        toast(ctx, msg)
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.dismantle_toast_fail, e.message ?: "") }
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.dismantle_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}