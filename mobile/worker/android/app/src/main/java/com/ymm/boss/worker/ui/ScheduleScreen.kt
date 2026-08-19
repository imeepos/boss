package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
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
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray

// 排期日历(对齐 docs/worker/schedule.html):月历忙碌日 + 今日排期 + 工时打卡
@Composable
fun ScheduleScreen(nav: NavHost) {
    var clockHint by remember { mutableStateOf("") }
    val state by loadOnce { ProfileApi.schedule() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    fun clock(type: String) {
        scope.launch {
            clockHint = try {
                val r = ProfileApi.clock(type)
                (if (type == "IN") "上班打卡成功" else "下班打卡成功") + "：" + r.optString("clockedAt")
            } catch (e: Exception) { "打卡失败：${e.message}" }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        when (val s = state) {
            is Load.Loading -> { TopBar("排期日历", onBack = { nav.pop() }); Loading() }
            is Load.Fail -> {
                TopBar("排期日历", onBack = { nav.pop() })
                Card(Modifier.padding(14.dp)) { Notice("排期加载失败：${s.message}", red = true) }
            }
            is Load.Ok -> {
                val d = s.data
                TopBar("排期日历", onBack = { nav.pop() }, action = d.optString("month", "-"))
                CalendarCard(d.optJSONArray("busyDays"))
                Card(Modifier.padding(12.dp)) {
                    val today = d.optJSONArray("today") ?: JSONArray()
                    SectionTitle("今日排期", more = "${today.length()} 单")
                    if (today.length() == 0) Empty("今日暂无排期")
                    for (i in 0 until today.length()) {
                        val it0 = today.optJSONObject(i)
                        val no = it0.optString("ticketNo")
                        Cell(title = it0.optString("time"),
                            desc = if (no.isEmpty()) it0.optString("address") else "$no · ${it0.optString("address")}",
                            onClick = if (no.isEmpty()) null else ({ nav.push(ticketScreen(no)) }))
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    SectionTitle("工时打卡")
                    PrimaryButton("上班打卡", modifier = Modifier.fillMaxWidth()) { clock("IN") }
                    Spacer(Modifier.height(8.dp))
                    PrimaryButton("下班打卡", modifier = Modifier.fillMaxWidth()) { clock("OUT") }
                    if (clockHint.isNotEmpty()) Notice(clockHint)
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun CalendarCard(busyDays: JSONArray?) {
    val busy = buildSet { for (i in 0 until (busyDays?.length() ?: 0)) add(busyDays?.optInt(i) ?: 0) }
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 6.dp)) {
            listOf("一", "二", "三", "四", "五", "六", "日").forEach {
                Text(it, fontSize = 12.sp, color = Muted, textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                    modifier = Modifier.weight(1f))
            }
        }
        var day = 1
        while (day <= 31) {
            Row(Modifier.fillMaxWidth()) {
                repeat(7) {
                    Box(Modifier.weight(1f).padding(3.dp), contentAlignment = Alignment.Center) {
                        if (day <= 31) {
                            val mark = day in busy
                            Text("$day", fontSize = 12.sp,
                                color = if (mark) Color.White else Ink,
                                fontWeight = if (mark) FontWeight.Bold else FontWeight.Normal,
                                modifier = Modifier.size(28.dp).let { m ->
                                    if (mark) m.background(Primary, CircleShape) else m
                                },
                                textAlign = androidx.compose.ui.text.style.TextAlign.Center)
                        }
                        day++
                    }
                }
            }
        }
    }
}