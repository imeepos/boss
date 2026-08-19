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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.AssetApi
import kotlinx.coroutines.launch

// 换机(对齐 docs/worker/replace.html):旧 EPC + 新 EPC 登记
@Composable
fun ReplaceScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { AssetApi.replace(no) }
    var oldEpc by remember { mutableStateOf("") }
    var newEpc by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("换件登记", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("换件信息加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("旧设备", d.optString("oldEpc", "-"))
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("旧 EPC")
            OutlinedTextField(value = oldEpc, onValueChange = { oldEpc = it },
                placeholder = { Text("扫描旧光猫标签") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel("新 EPC")
            OutlinedTextField(value = newEpc, onValueChange = { newEpc = it },
                placeholder = { Text("扫描新光猫标签") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton("确认换件", modifier = Modifier.fillMaxWidth()) {
                if (oldEpc.isBlank() || newEpc.isBlank()) { tip = "请填写旧/新 EPC"; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = AssetApi.submitReplace(no, oldEpc.trim(), newEpc.trim())
                        toast(ctx, r.optString("message", "换件登记完成，旧件转返修，新件沿用绑定"))
                        nav.pop()
                        ""
                    } catch (e: Exception) { "换件失败：${e.message}" }
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Row {
                ActionLink("退网登记") { nav.push(Screen.Retire(no)) }
                Spacer(Modifier.weight(1f))
                ActionLink("扫码绑定") { nav.push(Screen.Scan(no)) }
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