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
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 拆机作业(对齐 docs/worker/dismantle.html):扫码解绑释放端口
@Composable
fun DismantleScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    var epc by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("拆机作业", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> Card(Modifier.padding(12.dp)) {
                KvRow("工单号", s.data.optString("ticketNo"))
                KvRow("预绑定标签", s.data.optString("preBindTag", "-"))
                KvRow("分光器/端口", s.data.optString("splitterPort", "-"))
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("设备 EPC（扫码或手动输入）")
            OutlinedTextField(value = epc, onValueChange = { epc = it },
                placeholder = { Text("扫描光猫机身电子标签") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton("扫码解绑", modifier = Modifier.fillMaxWidth()) {
                if (epc.isBlank()) { tip = "请先输入或扫描 EPC"; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = AssetApi.dismantleScan(no, epc.trim())
                        val msg = if (r.optBoolean("unbound") && r.optBoolean("portReleased"))
                            "扫码解绑成功，端口已释放，可执行拆机" else r.optString("message", "解绑失败，请重试")
                        toast(ctx, msg)
                        ""
                    } catch (e: Exception) { "解绑失败：${e.message}" }
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Notice("拆机解绑后端口释放，旧件需按流程返库。")
        }
        Spacer(Modifier.height(12.dp))
    }
}