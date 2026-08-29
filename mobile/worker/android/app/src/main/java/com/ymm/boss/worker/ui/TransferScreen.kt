package com.ymm.boss.worker.ui

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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.TagRed
import kotlinx.coroutines.launch

// 转单(对齐 docs/worker/transfer.html)
@Composable
fun TransferScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val rSite = stringResource(R.string.transfer_reason_site)
    val rCust = stringResource(R.string.transfer_reason_customer)
    val rMat = stringResource(R.string.transfer_reason_material)
    val rOther = stringResource(R.string.transfer_reason_other)
    val reasons = listOf(rSite, rCust, rMat, rOther)
    val errBadId = stringResource(R.string.transfer_err_bad_id)
    val toastOk = stringResource(R.string.transfer_toast_ok)
    var reason by remember { mutableStateOf(reasons.first()) }
    var target by remember { mutableStateOf("") }
    var remark by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.transfer_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.transfer_load_fail), red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), d.optString("ticketNo"))
                    KvRow(stringResource(R.string.transfer_kv_slot), d.optString("scheduleSlot", "--"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.transfer_field_reason))
            OptionRow(reasons, reason) { reason = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.transfer_field_target))
            OutlinedTextField(value = target, onValueChange = { target = it },
                placeholder = { Text(stringResource(R.string.transfer_target_hint)) }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.transfer_field_remark))
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text(stringResource(R.string.transfer_remark_hint)) }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton(stringResource(R.string.transfer_submit), modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    tip = try {
                        val targetId = target.trim().takeIf { it.isNotEmpty() }?.toLongOrNull()
                        if (target.trim().isNotEmpty() && targetId == null) {
                            tip = errBadId
                            return@launch
                        }
                        val r = TicketApi.transfer(no, reason, targetId, remark.trim())
                        toast(ctx, r.optString("message", toastOk))
                        nav.pop()
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.transfer_toast_fail, e.message ?: "") }
                }
            }
            if (tip.isNotEmpty()) Text(tip, color = TagRed.fg,
                fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.transfer_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}

// 单选选项行(横排)
@Composable
fun OptionRow(options: List<String>, selected: String, onSelect: (String) -> Unit) {
    Column {
        options.forEach { opt ->
            Row(Modifier.fillMaxWidth().clickable { onSelect(opt) }.padding(vertical = 10.dp),
                horizontalArrangement = Arrangement.SpaceBetween) {
                Text(opt, fontSize = 14.sp, color = Ink)
                Text(if (opt == selected) "●" else "○", fontSize = 14.sp,
                    color = if (opt == selected) Primary
                    else androidx.compose.ui.graphics.Color(0xFFB0B3B8))
            }
        }
    }
}