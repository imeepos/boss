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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.ui.theme.Muted
import org.json.JSONArray

// 排障手册(对齐 docs/worker/help.html):FAQ/SOP 检索
@Composable
fun HelpScreen(nav: NavHost) {
    val state by loadOnce { MiscApi.faq(null) }
    var kw by remember { mutableStateOf("") }
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("排障手册", onBack = { nav.pop() }, action = "知识库")
        Card(Modifier.padding(12.dp)) {
            OutlinedTextField(value = kw, onValueChange = { kw = it },
                placeholder = { Text("检索 FAQ / SOP") }, singleLine = true,
                modifier = Modifier.fillMaxWidth())
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("高频 SOP")
            when (val s = state) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("排障手册加载失败：${s.message}", red = true)
                is Load.Ok -> {
                    val all = s.data.optJSONArray("items") ?: JSONArray()
                    val query = kw.trim()
                    val matched = if (query.isEmpty()) (0 until all.length()).toList()
                    else (0 until all.length()).filter { i ->
                        val it0 = all.optJSONObject(i)
                        it0.optString("title").contains(query) || it0.optString("summary").contains(query)
                    }
                    if (matched.isEmpty()) Empty("未找到相关 SOP")
                    for (i in matched) {
                        val it0 = all.optJSONObject(i)
                        Cell(
                            title = it0.optString("title"),
                            desc = it0.optString("summary"),
                            onClick = { toast(ctx, it0.optString("content", it0.optString("summary"))) },
                            right = { Text(">", fontSize = 14.sp, color = Muted) },
                        )
                    }
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            Notice("知识库多语言（中/英/菲）可检索，装维与客服共用。")
        }
        Spacer(Modifier.height(12.dp))
    }
}