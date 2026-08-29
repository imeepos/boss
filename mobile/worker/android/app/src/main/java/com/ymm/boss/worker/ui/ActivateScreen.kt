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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ScanApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.TagBlue
import kotlinx.coroutines.launch
import org.json.JSONObject

// 激活(对齐 docs/worker/activate.html):LOID + 状态 + 重新激活
@Composable
fun ActivateScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { ScanApi.activation(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val toastOk = stringResource(R.string.act_toast_ok)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.act_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.act_load_fail), red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), d.optString("ticketNo"))
                    KvRow(stringResource(R.string.act_kv_loid), d.optString("loid", "-"))
                    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp),
                        horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(stringResource(R.string.act_kv_status), fontSize = 13.sp, color = Muted)
                        StatusTag(d.optString("statusLabel", "-"), d.optString("status"))
                    }
                    KvRow(stringResource(R.string.act_kv_last_try), d.optString("lastTry", "-"))
                }
                Card(Modifier.padding(12.dp)) {
                    Box(Modifier.fillMaxWidth()
                        .clip(RoundedCornerShape(16.dp))
                        .background(TagBlue.bg)
                        .padding(vertical = 32.dp), contentAlignment = Alignment.Center) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Text("↻", fontSize = 40.sp, color = Primary, fontWeight = FontWeight.Bold)
                            Text(stringResource(R.string.act_prompt_fail), fontSize = 13.sp, color = Muted)
                            Text(stringResource(R.string.act_prompt_retry), fontSize = 12.sp, color = Muted)
                        }
                    }
                    Text(stringResource(R.string.act_btn_retry), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Color.White,
                        modifier = Modifier.fillMaxWidth()
                            .padding(top = 12.dp)
                            .clip(RoundedCornerShape(10.dp))
                            .background(Success)
                            .clickable {
                                scope.launch {
                                    try {
                                        val r = ScanApi.activate(no)
                                        toast(ctx, r.optString("message", toastOk))
                                        nav.push(Screen.Report(no))
                                    } catch (e: Exception) { toast(ctx, ctx.getString(R.string.act_toast_fail, e.message ?: "")) }
                                }
                            }
                            .padding(vertical = 12.dp),
                        textAlign = TextAlign.Center)
                }
                Card(Modifier.padding(12.dp)) {
                    Notice(stringResource(R.string.act_notice))
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}