package com.ymm.boss.worker.ui

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.MiscApi
import kotlinx.coroutines.launch
import org.json.JSONArray

// 联系调度(对齐 docs/worker/service.html):会话流 + 拨号 + 转单入口
@Composable
fun ServiceScreen(nav: NavHost) {
    var refresh by remember { mutableStateOf(0) }
    var content by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    val state by loadOnce(refresh) { MiscApi.serviceMessages() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val placeholder = stringResource(R.string.svc_hint)
    val svcPlaceholder = stringResource(R.string.svc_placeholder)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.svc_btn_contact), onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice(ctx.getString(R.string.svc_load_fail, s.message ?: ""), red = true)
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    if (items.length() == 0) Empty(stringResource(R.string.svc_empty))
                    for (i in 0 until items.length()) {
                        val m = items.optJSONObject(i)
                        val mine = m.optString("from") == "worker"
                        Row(Modifier.fillMaxWidth().padding(vertical = 4.dp),
                            horizontalArrangement = if (mine)
                                androidx.compose.foundation.layout.Arrangement.End
                            else androidx.compose.foundation.layout.Arrangement.Start) {
                            Text(m.optString("content"), fontSize = 13.sp, color = Color.White,
                                modifier = Modifier
                                    .background(
                                        if (mine) Color(0xFF1677FF) else Color(0xFF8C8C8C),
                                        RoundedCornerShape(10.dp),
                                    )
                                    .padding(horizontal = 12.dp, vertical = 8.dp))
                        }
                    }
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            OutlinedTextField(
                value = content,
                onValueChange = { content = it },
                placeholder = { Text(placeholder) },
                minLines = 2,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))
            PrimaryButton(stringResource(R.string.svc_btn_send), enabled = !sending && content.isNotBlank(), modifier = Modifier.fillMaxWidth()) {
                sending = true
                scope.launch {
                    try {
                        MiscApi.sendServiceMessage(content.trim())
                        content = ""
                        refresh++
                    } catch (e: Exception) {
                        toast(ctx, ctx.getString(R.string.svc_send_fail, e.message ?: ""))
                    }
                    sending = false
                }
            }
        }
        Row(Modifier.fillMaxWidth().padding(12.dp),
            horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(10.dp)) {
            PrimaryButton(stringResource(R.string.svc_btn_call), modifier = Modifier.weight(1f)) {
                ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:10086")))
            }
            PrimaryButton(stringResource(R.string.svc_title), modifier = Modifier.weight(1f)) { toast(ctx, svcPlaceholder) }
        }
        Spacer(Modifier.height(12.dp))
    }
}