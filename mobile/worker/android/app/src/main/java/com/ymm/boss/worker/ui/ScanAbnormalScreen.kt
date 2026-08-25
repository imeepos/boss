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

// 扫码异常上报(对齐 docs/worker/scanabnormal.html):类型+实物标签+描述,禁止手工绕过
@Composable
fun ScanAbnormalScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val ctx = LocalContext.current
    val toastOk = stringResource(R.string.sa_toast_ok)
    val abnTypes = listOf(
        stringResource(R.string.sa_abn_damaged),
        stringResource(R.string.sa_abn_mismatch),
        stringResource(R.string.sa_abn_missing),
        stringResource(R.string.sa_abn_device_fail),
    )
    var cat by remember { mutableStateOf(abnTypes[0]) }
    var epc by remember { mutableStateOf("") }
    var desc by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.sa_title), onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val s = info) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice(stringResource(R.string.sa_load_fail), red = true)
                is Load.Ok -> {
                    KvRow(stringResource(R.string.td_ticket_no), s.data.optString("ticketNo"))
                    KvRow(stringResource(R.string.sa_kv_scene), stringResource(R.string.sa_scene_scan_abn))
                    KvRow(stringResource(R.string.sa_kv_prebind), s.data.optString("preBindTag", "-"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.sa_field_type))
            OptionRow(abnTypes, cat) { cat = it }
            FieldLabel(stringResource(R.string.sa_field_actual))
            OutlinedTextField(value = epc, onValueChange = { epc = it },
                placeholder = { Text(stringResource(R.string.sa_actual_hint)) }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.sa_field_desc))
            OutlinedTextField(value = desc, onValueChange = { desc = it },
                placeholder = { Text(stringResource(R.string.sa_desc_hint)) }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton(stringResource(R.string.sa_btn_photo), modifier = Modifier.fillMaxWidth()) { nav.push(Screen.Photo(no)) }
            Spacer(Modifier.height(8.dp))
            PrimaryButton(stringResource(R.string.sa_btn_submit), modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    tip = try {
                        val r = ScanApi.abnormal(no, JSONObject().apply {
                            put("category", cat); put("epc", epc.trim()); put("description", desc)
                        })
                        toast(ctx, r.optString("message", toastOk))
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.sa_toast_fail, e.message ?: "") }
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.sa_abn_notice), red = true)
        }
        Spacer(Modifier.height(12.dp))
    }
}