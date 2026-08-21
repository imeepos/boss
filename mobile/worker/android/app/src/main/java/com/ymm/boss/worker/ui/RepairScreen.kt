package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch

/**
 * 修复上报表单(从原 RepairScreen 拆出,仅保留报障结果表单):
 * - 字段:结果(FIXED/UNFIXED) + 备注
 * - 提交:POST /tickets/{ticketNo}/repair-report,响应 reviewPassed 决定关闭/回退提示
 */
@Composable
fun RepairReportScreen(nav: NavHost, no: String) {
    var result by remember { mutableStateOf("FIXED") }
    var remark by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("修复上报", onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            SectionTitle("修复结果")
            OptionRow(listOf("已恢复", "未恢复"),
                if (result == "FIXED") "已恢复" else "未恢复") {
                result = if (it == "已恢复") "FIXED" else "UNFIXED"
            }
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text("补充现场处理说明") }, minLines = 2,
                modifier = Modifier.fillMaxWidth())
        }
        Card(Modifier.padding(12.dp)) {
            Row(modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                SecondaryBtn("取消", Modifier.weight(1f)) { nav.pop() }
                PrimaryBtn("提交上报", Modifier.weight(1f)) {
                    scope.launch {
                        try {
                            val r = TicketApi.repairReport(no, result, remark.trim())
                            toast(ctx, if (r.optBoolean("reviewPassed"))
                                "网络已恢复，工单关闭，系统自动复核通过"
                            else "结果已上报，系统复核中")
                            nav.pop()
                        } catch (_: Exception) { toast(ctx, "上报失败，请重试。") }
                    }
                }
            }
        }
        Notice("上报修复后系统自动复核网络是否恢复；未恢复将回「处理中」，恢复则关闭并触发满意度回访。")
        Spacer(Modifier.height(12.dp))
    }
}

// 报障工单(TKT/EMG 前缀)的详情落地页直接复用 TicketDetailScreen(specs §6)
@Composable
fun RepairScreen(nav: NavHost, no: String) = TicketDetailScreen(nav, no)

// 主按钮(蓝实底)
@Composable
private fun PrimaryBtn(text: String, modifier: Modifier, onClick: () -> Unit) {
    Box(modifier = modifier.background(Color(0xFF086CF5), RoundedCornerShape(10.dp))
        .clickable { onClick() }.padding(vertical = 12.dp),
        contentAlignment = Alignment.Center) {
        Text(text, color = Color.White, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
    }
}

// 次按钮(白底蓝字)
@Composable
private fun SecondaryBtn(text: String, modifier: Modifier, onClick: () -> Unit) {
    Box(modifier = modifier.background(Color.White, RoundedCornerShape(10.dp))
        .clickable { onClick() }.padding(vertical = 12.dp),
        contentAlignment = Alignment.Center) {
        Text(text, color = Primary, fontSize = 15.sp, fontWeight = FontWeight.Medium)
    }
}