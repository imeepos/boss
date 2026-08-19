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

private val ABN_TYPES = listOf(
    "标签无法读取（污损/损坏）",
    "扫码结果与预绑定不一致",
    "光猫无标签/被替换",
    "扫码枪/手机 NFC 故障",
)

// 扫码异常上报(对齐 docs/worker/scanabnormal.html):类型+实物标签+描述,禁止手工绕过
@Composable
fun ScanAbnormalScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    var cat by remember { mutableStateOf(ABN_TYPES[0]) }
    var epc by remember { mutableStateOf("") }
    var desc by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("扫码异常上报", onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val s = info) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("工单加载失败，请重试。", red = true)
                is Load.Ok -> {
                    KvRow("工单号", s.data.optString("ticketNo"))
                    KvRow("场景", "扫码绑定异常")
                    KvRow("预绑定标签", s.data.optString("preBindTag", "-"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("异常类型")
            OptionRow(ABN_TYPES, cat) { cat = it }
            FieldLabel("实物标签号（如可读）")
            OutlinedTextField(value = epc, onValueChange = { epc = it },
                placeholder = { Text("选填，手工录入将强制标记") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel("现场描述")
            OutlinedTextField(value = desc, onValueChange = { desc = it },
                placeholder = { Text("描述现场异常情况") }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton("拍照取证", modifier = Modifier.fillMaxWidth()) { nav.push(Screen.Photo(no)) }
            Spacer(Modifier.height(8.dp))
            PrimaryButton("提交异常", modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    tip = try {
                        val r = ScanApi.abnormal(no, JSONObject().apply {
                            put("category", cat); put("epc", epc.trim()); put("description", desc)
                        })
                        toast(ctx, r.optString("message", "异常已上报"))
                        ""
                    } catch (e: Exception) { "提交失败：${e.message}" }
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Notice("扫码是绑定强制节点：异常须上报并转调度复核，禁止以手工录入绕过扫码。", red = true)
        }
        Spacer(Modifier.height(12.dp))
    }
}