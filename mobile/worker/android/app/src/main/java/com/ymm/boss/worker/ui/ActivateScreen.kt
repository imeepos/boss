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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.ScanApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONObject

// 激活(对齐 docs/worker/activate.html):LOID + 状态 + 重新激活
@Composable
fun ActivateScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { ScanApi.activation(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("激活", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("激活信息加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("LOID", d.optString("loid", "-"))
                    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp),
                        horizontalArrangement = Arrangement.SpaceBetween) {
                        Text("状态", fontSize = 13.sp, color = Muted)
                        StatusTag(d.optString("statusLabel", "-"), d.optString("status"))
                    }
                    KvRow("上次尝试", d.optString("lastTry", "-"))
                }
                Card(Modifier.padding(12.dp)) {
                    Box(Modifier.fillMaxWidth()
                        .clip(RoundedCornerShape(16.dp))
                        .background(Color(0xFFE6F4FF))
                        .padding(vertical = 32.dp), contentAlignment = Alignment.Center) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Text("↻", fontSize = 40.sp, color = Primary, fontWeight = FontWeight.Bold)
                            Text("激活回调未返回成功", fontSize = 13.sp, color = Muted)
                            Text("请确认光猫在线后重试激活", fontSize = 12.sp, color = Muted)
                        }
                    }
                    Text("重新激活", fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Color.White,
                        modifier = Modifier.fillMaxWidth()
                            .padding(top = 12.dp)
                            .clip(RoundedCornerShape(10.dp))
                            .background(Color(0xFF52C41A))
                            .clickable {
                                scope.launch {
                                    try {
                                        val r = ScanApi.activate(no)
                                        toast(ctx, r.optString("message", "激活成功"))
                                        nav.push(Screen.Report(no))
                                    } catch (e: Exception) { toast(ctx, "激活失败：${e.message}") }
                                }
                            }
                            .padding(vertical = 12.dp),
                        textAlign = TextAlign.Center)
                }
                Card(Modifier.padding(12.dp)) {
                    Notice("激活失败不丢数据，可反复重试；连续失败将自动转告警派单。")
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}