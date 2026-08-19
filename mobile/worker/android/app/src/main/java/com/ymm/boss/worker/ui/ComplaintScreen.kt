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
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 报障(对齐 docs/worker/complaint.html)
@Composable
fun ComplaintScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val cats = listOf("网络不通", "网速不达标", "服务态度", "其他")
    var category by remember { mutableStateOf(cats.first()) }
    var content by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("报障登记", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("工单加载失败，请刷新重试。", red = true) }
            is Load.Ok -> Card(Modifier.padding(12.dp)) {
                KvRow("工单号", s.data.optString("ticketNo"))
                KvRow("客户", "${s.data.optString("customerName")} · ${s.data.optString("customerPhoneMasked")}")
            }
        }
        Card(Modifier.padding(12.dp)) {
            FieldLabel("投诉分类")
            OptionRow(cats, category) { category = it }
            Spacer(Modifier.height(12.dp))
            FieldLabel("问题描述")
            OutlinedTextField(value = content, onValueChange = { content = it },
                placeholder = { Text("请描述遇到的问题") }, minLines = 3,
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            PrimaryButton("提交报障", modifier = Modifier.fillMaxWidth()) {
                if (content.isBlank()) { tip = "请填写问题描述"; return@PrimaryButton }
                scope.launch {
                    tip = try {
                        val r = TicketApi.complaint(no, category, content.trim())
                        toast(ctx, r.optString("message", "报障已提交"))
                        nav.pop()
                        ""
                    } catch (e: Exception) { "报障失败：${e.message}" }
                }
            }
            if (tip.isNotEmpty()) Text(tip, color = androidx.compose.ui.graphics.Color(0xFFCF1322),
                fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
        }
        Spacer(Modifier.height(12.dp))
    }
}