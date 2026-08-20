package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ServiceApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

/** 对应草稿 docs/user/service.html:智能客服对话 + 常见问题。 */
private data class Msg(val fromMe: Boolean, val text: String)

@Composable
fun ServiceScreen(nav: Nav) {
    val msgs = remember { mutableStateListOf(Msg(false, "您好，我是智能客服小维。可为您解答账单、套餐、报障等问题，输入「人工」可转接人工坐席。")) }
    var faqs by remember { mutableStateOf(emptyList<JSONObject>()) }
    var input by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(nav.refreshTick) {
        try { faqs = ServiceApi.faq().optJSONArray("items").toList() } catch (e: Exception) { /* 保留骨架 */ }
    }

    Column {
        TopBar("在线客服", onBack = { nav.pop() }, action = "帮助中心", onAction = { nav.push(Route.Help) })
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).imePadding()) {
            AppCard {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Text("智能客服", fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink, modifier = Modifier.weight(1f))
                    Tag("在线", Palette.success)
                }
                msgs.forEach { Bubble(it) }
            }
            AppCard {
                CardTitle("常见问题", "更多") { nav.push(Route.Help) }
                if (faqs.isEmpty()) EmptyState("暂无常见问题")
                faqs.forEach { f -> CellRow(title = f.optString("question"), right = { Text("›", color = Palette.subtle) }) }
            }
        }
        ChatInput(input, { input = it }) { send(scope, msgs, input) { input = "" } }
    }
}

@Composable
private fun Bubble(m: Msg) {
    Row(Modifier.fillMaxWidth().padding(top = 12.dp), horizontalArrangement = if (m.fromMe) Arrangement.End else Arrangement.Start) {
        Text(
            m.text, fontSize = 13.sp,
            color = if (m.fromMe) Color.White else Palette.ink,
            modifier = Modifier
                .fillMaxWidth(0.78f)
                // 对方气泡底色走 Palette.line 浅色语义,暗色主题下同色系不刺眼
                .background(if (m.fromMe) Palette.primary else Palette.line, RoundedCornerShape(12.dp))
                .padding(horizontal = 12.dp, vertical = 10.dp),
        )
    }
}

@Composable
private fun ChatInput(value: String, onChange: (String) -> Unit, onSend: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
        OutlinedTextField(value = value, onValueChange = onChange, placeholder = { Text("输入你的问题…") }, singleLine = true, modifier = Modifier.weight(1f))
        Button(onClick = onSend, colors = ButtonDefaults.buttonColors(containerColor = Palette.primary)) { Text("发送") }
    }
}

private fun send(scope: kotlinx.coroutines.CoroutineScope, msgs: MutableList<Msg>, text: String, onSent: () -> Unit) {
    val msg = text.trim()
    if (msg.isEmpty()) return
    msgs.add(Msg(true, msg))
    onSent()
    scope.launch {
        try {
            val r = ServiceApi.chat(msg)
            msgs.add(Msg(false, r.optString("reply")))
            if (r.optBoolean("toHuman")) msgs.add(Msg(false, "正在为您转接人工坐席…"))
        } catch (e: Exception) { msgs.add(Msg(false, "网络异常，请稍后重试")) }
    }
}

internal fun JSONArray?.toList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
