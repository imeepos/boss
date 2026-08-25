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

// 报障(对齐 docs/worker/complaint.html)
@Composable
fun ComplaintScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val catNet = stringResource(R.string.complaint_cat_network)
    val catSpeed = stringResource(R.string.complaint_cat_speed)
    val catSvc = stringResource(R.string.complaint_cat_service)
    val catOther = stringResource(R.string.complaint_cat_other)
    val cats = listOf(catNet, catSpeed, catSvc, catOther)
    val errBlank = stringResource(R.string.complaint_err_blank)
    val toastOk = stringResource(R.string.complaint_toast_ok)
    var category by remember { mutableStateOf(cats.first()) }
    var content by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.complaint_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.complaint_load_fail), red = true) }
            is Load.Ok -> Card(Modifier.padding(12.dp)) {
                KvRow(stringResource(R.string.td_ticket_no), s.data.optString("ticketNo"))
                KvRow(stringResource(R.string.td_customer), "${s.data.optString("customerName")} · ${s.data.optString("customerPhoneMasked")}")
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel(stringResource(R.string.complaint_field_category))
            OptionRow(cats, category) { category = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.complaint_field_content))
            OutlinedTextField(value = content, onValueChange = { content = it },
                placeholder = { Text(stringResource(R.string.complaint_content_hint)) }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton(stringResource(R.string.complaint_submit), modifier = Modifier.fillMaxWidth()) {
                if (content.isBlank()) { tip = errBlank; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = TicketApi.complaint(no, category, content.trim())
                        toast(ctx, r.optString("message", toastOk))
                        nav.pop()
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.complaint_toast_fail, e.message ?: "") }
                }
            }
            if (tip.isNotEmpty()) Text(tip, color = androidx.compose.ui.graphics.Color(0xFFCF1322),
                fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
        }
        Spacer(Modifier.height(12.dp))
    }
}