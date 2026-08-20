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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray

private val WORK_TYPES = listOf("高空作业（爬杆/登高）", "带电作业", "密闭/井道作业", "常规入户")
private val CHECK_ITEMS = listOf(
    "安全帽 / 绝缘鞋已佩戴",
    "安全带 / 防坠措施已就位",
    "作业区域已围挡警示",
    "已告知客户作业风险",
)

// 安全作业上报(对齐 docs/worker/safety.html):作业类型 + 确认清单
@Composable
fun SafetyScreen(nav: NavHost) {
    var workType by remember { mutableStateOf(WORK_TYPES[0]) }
    var checked by remember { mutableStateOf(setOf<String>()) }
    var tip by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("安全作业上报", onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            SectionTitle("作业类型")
            OptionRow(WORK_TYPES, workType) { workType = it }
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("安全确认清单")
            CHECK_ITEMS.forEach { item ->
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
            PrimaryButton("提交安全确认", enabled = !submitting, modifier = Modifier.fillMaxWidth()) {
                if (checked.isEmpty()) {
                    tip = "请先完成至少一项安全确认"
                    return@PrimaryButton
                }
                submitting = true
                scope.launch {
                    tip = try {
                        val list = JSONArray()
                        CHECK_ITEMS.filter { it in checked }.forEach { list.put(it) }
                        val r = MiscApi.safetyCheck(workType, list)
                        toast(ctx, r.optString("message", "安全确认已上报留痕"))
                        ""
                    } catch (e: Exception) { "上报失败：${e.message}" }
                    submitting = false
                }
            }
            if (tip.isNotEmpty()) Notice(tip, red = true)
        }
        Card(Modifier.padding(12.dp)) {
            Notice("高风险作业须先完成安全确认方可开始，确认记录留痕可追溯。")
        }
        Spacer(Modifier.height(12.dp))
    }
}