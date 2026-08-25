package com.ymm.boss.worker.ui

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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray

// 安全作业上报(对齐 docs/worker/safety.html):作业类型 + 确认清单
@Composable
fun SafetyScreen(nav: NavHost) {
    val ctx = LocalContext.current
    val workTypes = listOf(
        stringResource(R.string.safety_type_heights),
        stringResource(R.string.safety_type_live),
        stringResource(R.string.safety_type_confined),
        stringResource(R.string.safety_type_general),
    )
    val checkItems = listOf(
        stringResource(R.string.safety_item_helmet),
        stringResource(R.string.safety_item_harness),
        stringResource(R.string.safety_item_barrier),
        stringResource(R.string.safety_item_aware),
    )
    val errEmpty = stringResource(R.string.safety_err_empty)
    val toastOk = stringResource(R.string.safety_submit_ok)
    var workType by remember { mutableStateOf(workTypes[0]) }
    var checked by remember { mutableStateOf(setOf<String>()) }
    var tip by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.safety_title), onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.safety_kv_type))
            OptionRow(workTypes, workType) { workType = it }
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.safety_section_checks))
            checkItems.forEach { item ->
                val on = item in checked
                Row(Modifier.fillMaxWidth().clickable {
                    checked = if (on) checked - item else checked + item
                }.padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text(if (on) "☑" else "☐", fontSize = 16.sp,
                        color = if (on) Primary else androidx.compose.ui.graphics.Color(0xFFB8BCC2))
                    Text(item, fontSize = 14.sp, modifier = Modifier.padding(start = 10.dp))
                }
            }
            Spacer(Modifier.height(8.dp))
            PrimaryButton(stringResource(R.string.safety_submit), enabled = !submitting, modifier = Modifier.fillMaxWidth()) {
                if (checked.isEmpty()) {
                    tip = errEmpty
                    return@PrimaryButton
                }
                submitting = true
                scope.launch {
                    tip = try {
                        val list = JSONArray()
                        checkItems.filter { it in checked }.forEach { list.put(it) }
                        val r = MiscApi.safetyCheck(workType, list)
                        toast(ctx, r.optString("message", toastOk))
                        ""
                    } catch (e: Exception) { ctx.getString(R.string.safety_submit_fail, e.message ?: "") }
                    submitting = false
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.safety_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}