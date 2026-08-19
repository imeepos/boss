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
import androidx.compose.ui.unit.dp
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

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("完工上报", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("客户", "${d.optString("customerName")}(${d.optString("customerPhoneMasked")})")
                    KvRow("地址", d.optString("address"))
                }
            }
        }
        if (err.isNotEmpty()) Card(Modifier.padding(12.dp)) { Notice(err, red = true) }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("备注")
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text("填写完工备注（可选）") }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
        }
        Card(Modifier.padding(12.dp)) {
            PrimaryButton("提交上报", enabled = !busy, modifier = Modifier.fillMaxWidth()) {
                busy = true
                scope.launch {
                    try {
                        val r = ScanApi.submitReport(no, remark.trim())
                        toast(ctx, r.optString("message", "上报成功"))
                        nav.pop()
                    } catch (e: Exception) {
                        err = "上报失败：${e.message}"
                        busy = false
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}