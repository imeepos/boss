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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
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
    val fixedLabel = stringResource(R.string.repair_result_fixed)
    val unfixedLabel = stringResource(R.string.repair_result_unfixed)
    val options = listOf(fixedLabel, unfixedLabel)
    val cancelLabel = stringResource(R.string.btn_cancel)
    val toastPassed = stringResource(R.string.repair_toast_passed)
    val toastPending = stringResource(R.string.repair_toast_pending)
    val toastFail = stringResource(R.string.repair_toast_fail)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.repair_title), onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.repair_result_title))
            OptionRow(options, if (result == "FIXED") fixedLabel else unfixedLabel) {
                result = if (it == fixedLabel) "FIXED" else "UNFIXED"
            }
            OutlinedTextField(value = remark, onValueChange = { remark = it },
                placeholder = { Text(stringResource(R.string.repair_remark_hint)) }, minLines = 2,
                modifier = Modifier.fillMaxWidth())
        }
        Card(Modifier.padding(12.dp)) {
            Row(modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                SecondaryBtn(cancelLabel, Modifier.weight(1f)) { nav.pop() }
                PrimaryBtn(stringResource(R.string.repair_submit), Modifier.weight(1f)) {
                    scope.launch {
                        try {
                            val r = TicketApi.repairReport(no, result, remark.trim())
                            toast(ctx, if (r.optBoolean("reviewPassed")) toastPassed else toastPending)
                            nav.pop()
                        } catch (_: Exception) { toast(ctx, toastFail) }
                    }
                }
            }
        }
        Notice(stringResource(R.string.repair_notice))
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