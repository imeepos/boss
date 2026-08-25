package com.ymm.boss.worker.ui

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
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.AssetApi
import kotlinx.coroutines.launch

// 换机(对齐 docs/worker/replace.html):旧 EPC + 新 EPC 登记
@Composable
fun ReplaceScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { AssetApi.replace(no) }
    val errBlank = stringResource(R.string.replace_err_blank_epc)
    val toastOk = stringResource(R.string.replace_toast_ok)
    var oldEpc by remember { mutableStateOf("") }
    var newEpc by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.replace_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.replace_load_fail), red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), d.optString("ticketNo"))
                    KvRow(stringResource(R.string.replace_kv_old_device), d.optString("oldEpc", "-"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.replace_field_old_epc))
            OutlinedTextField(value = oldEpc, onValueChange = { oldEpc = it },
                placeholder = { Text(stringResource(R.string.replace_old_epc_hint)) }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.replace_field_new_epc))
            OutlinedTextField(value = newEpc, onValueChange = { newEpc = it },
                placeholder = { Text(stringResource(R.string.replace_new_epc_hint)) }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton(stringResource(R.string.replace_submit), modifier = Modifier.fillMaxWidth()) {
                if (oldEpc.isBlank() || newEpc.isBlank()) { tip = errBlank; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = AssetApi.submitReplace(no, oldEpc.trim(), newEpc.trim())
                        toast(ctx, r.optString("message", toastOk))
                        nav.pop()
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.replace_toast_fail, e.message ?: "") }
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Row {
                ActionLink(stringResource(R.string.replace_action_retire)) { nav.push(Screen.Retire(no)) }
                Spacer(Modifier.weight(1f))
                ActionLink(stringResource(R.string.replace_action_scan)) { nav.push(Screen.Scan(no)) }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun ActionLink(text: String, onClick: () -> Unit) {
    Text(text, fontSize = 14.sp, color = androidx.compose.ui.graphics.Color(0xFF1677FF),
        modifier = Modifier.padding(vertical = 8.dp).clickable { onClick() })
}