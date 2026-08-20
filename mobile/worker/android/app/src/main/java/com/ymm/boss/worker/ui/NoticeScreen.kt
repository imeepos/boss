package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.MiscApi
import com.ymm.boss.worker.ui.theme.Muted
import org.json.JSONArray

// 服务公告(对齐 docs/worker/notice.html):公告列表
@Composable
fun NoticeScreen(nav: NavHost) {
    val state by loadOnce { MiscApi.notices() }
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("服务公告", onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("公告加载失败：${s.message}", red = true)
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    if (items.length() == 0) Empty("暂无公告")
                    for (i in 0 until items.length()) {
                        val n = items.optJSONObject(i)
                        Cell(
                            title = n.optString("title"),
                            desc = listOfNotNull(
                                n.optString("publishedAt").takeIf { it.isNotEmpty() },
                                n.optString("category").takeIf { it.isNotEmpty() },
                            ).joinToString(" · "),
                            onClick = { toast(ctx, n.optString("content", "无详情")) },
                            right = { Text(">", fontSize = 14.sp, color = Muted) },
                        )
                    }
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            Notice("公告与消息中心联动，重要通告同步推送到师傅端。")
        }
        Spacer(Modifier.height(12.dp))
    }
}