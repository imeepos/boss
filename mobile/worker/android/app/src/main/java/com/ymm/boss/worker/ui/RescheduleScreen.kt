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
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 改约(对齐 docs/worker/reschedule.html)
@Composable
fun RescheduleScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val slots = listOf("09:00-11:00", "14:00-16:00", "16:00-18:00")
    val rReq = stringResource(R.string.reschedule_reason_customer_request)
    val rNoShow = stringResource(R.string.reschedule_reason_no_show)
    val rSched = stringResource(R.string.reschedule_reason_schedule_conflict)
    val rWx = stringResource(R.string.reschedule_reason_weather_traffic)
    val reasons = listOf(rReq, rNoShow, rSched, rWx)
    val errNoDate = stringResource(R.string.reschedule_err_no_date)
    val toastOk = stringResource(R.string.reschedule_toast_ok)
    var date by remember { mutableStateOf("") }
    var slot by remember { mutableStateOf(slots.first()) }
    var reason by remember { mutableStateOf(reasons.first()) }
    var remark by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.reschedule_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.reschedule_load_fail), red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), d.optString("ticketNo"))
                    KvRow(stringResource(R.string.td_customer), "${d.optString("customerName")} · ${d.optString("customerPhoneMasked")}")
                    KvRow(stringResource(R.string.reschedule_kv_original), d.optString("scheduleSlot", "--"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.reschedule_field_date))
            OutlinedTextField(value = date, onValueChange = { date = it },
                placeholder = { Text(stringResource(R.string.reschedule_date_hint)) }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.reschedule_field_slot))
            OptionRow(slots, slot) { slot = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.reschedule_field_reason))
            OptionRow(reasons, reason) { reason = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.reschedule_field_remark))
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text(stringResource(R.string.reschedule_remark_hint)) }, minLines = 2,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton(stringResource(R.string.reschedule_submit), modifier = Modifier.fillMaxWidth()) {
                if (date.isBlank()) { tip = errNoDate; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = TicketApi.reschedule(no, date.trim(), slot, reason, remark.trim())
                        toast(ctx, r.optString("message", toastOk))
                        nav.pop()
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.reschedule_toast_fail, e.message ?: "") }
                }
            }
            if (tip.isNotEmpty()) Text(tip, color = androidx.compose.ui.graphics.Color(0xFFCF1322),
                fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.reschedule_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}