package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.MiscApi
import kotlinx.coroutines.launch
import org.json.JSONArray

// 消息中心(对齐 docs/worker/messages.html):全部已读 + 清空已读
@Composable
fun MessagesScreen(nav: NavHost) {
    var refresh by remember { mutableStateOf(0) }
    val state by loadOnce(refresh) { MiscApi.messages() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("消息中心", onBack = { nav.pop() }, action = "全部已读", onAction = {
            scope.launch {
                try { toast(ctx, MiscApi.readAll().optString("message", "全部标记已读")); refresh++ }
                catch (e: Exception) { toast(ctx, "操作失败：${e.message}") }
            }
        })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("消息加载失败：${s.message}", red = true)
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    if (items.length() == 0) Empty("暂无消息")
                    for (i in 0 until items.length()) {
                        val m = items.optJSONObject(i)
                        MsgBanner(m.optString("level", "info"),
                            m.optString("title"), m.optString("content"), m.optString("sentAt"))
                        Spacer(Modifier.height(8.dp))
                    }
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            PrimaryButton("清空已读", modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    try { toast(ctx, MiscApi.clear().optString("message", "已清空已读消息")); refresh++ }
                    catch (e: Exception) { toast(ctx, "操作失败：${e.message}") }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun MsgBanner(level: String, title: String, content: String, sentAt: String) {
    val bg = when (level) {
        "warn" -> Color(0xFFFFF7E6)
        "error" -> Color(0xFFFFF1F0)
        else -> Color(0xFFF0F5FF)
    }
    val fg = when (level) {
        "warn" -> Color(0xFFD46B08)
        "error" -> Color(0xFFCF1322)
        else -> Color(0xFF0958D9)
    }
    Column(Modifier.fillMaxWidth().background(bg, RoundedCornerShape(10.dp)).padding(12.dp)) {
        Text("$title · $sentAt", fontSize = 13.sp, color = fg)
        Text(content, fontSize = 13.sp, color = Color(0xFF333333), modifier = Modifier.padding(top = 4.dp))
    }
}