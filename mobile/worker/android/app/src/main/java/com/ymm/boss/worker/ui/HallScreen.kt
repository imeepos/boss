package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.HallApi
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.launch
import org.json.JSONArray

// 任务池(对齐 docs/worker/hall.html):跨区公开可抢工单,先到先得
@Composable
fun HallScreen(nav: NavHost) {
    var refresh by remember { mutableStateOf(0) }
    var grabbed by remember { mutableStateOf(setOf<String>()) }
    val state by loadOnce(refresh) { HallApi.list() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    fun grab(no: String) {
        scope.launch {
            try {
                val r = HallApi.grab(no)
                toast(ctx, r.optString("message", "抢单成功"))
                grabbed = grabbed + no
                refresh++
            } catch (e: Exception) { toast(ctx, "抢单失败，请重试。") }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("任务池", onBack = { nav.pop() }, action = "刷新", onAction = { refresh++ })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> { SectionTitle("可抢工单 · 加载中…"); Loading() }
                is Load.Fail -> { SectionTitle("可抢工单"); Notice("任务池加载失败，请刷新重试。", red = true) }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    SectionTitle("可抢工单 (${items.length()} 单)")
                    if (items.length() == 0) Empty("暂无可抢工单")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val no = it0.optString("ticketNo")
                        val dist = if (it0.isNull("distanceKm")) "" else " · 距您 ${it0.optDouble("distanceKm")}km"
                        val isGrabbed = grabbed.contains(no)
                        Cell(
                            title = "$no · ${it0.optString("typeLabel")}",
                            desc = it0.optString("address") + dist,
                            onClick = if (isGrabbed) ({ nav.push(ticketScreen(no)) }) else null,
                        ) {
                            Text(if (isGrabbed) "已抢" else "抢单", fontSize = 13.sp,
                                color = if (isGrabbed) Success else Color.White,
                                modifier = Modifier
                                    .clip(RoundedCornerShape(8.dp))
                                    .background(if (isGrabbed) Color.Transparent else Primary)
                                    .clickable(enabled = !isGrabbed) { grab(no) }
                                    .padding(horizontal = 12.dp, vertical = 6.dp))
                        }
                    }
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            Notice("任务池按区域与技能公开，先到先得；抢单后进入「我的工单」。")
        }
        Spacer(Modifier.height(12.dp))
    }
}