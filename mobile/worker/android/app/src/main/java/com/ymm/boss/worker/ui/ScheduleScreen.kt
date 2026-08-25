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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray

// 排期日历(对齐 docs/worker/schedule.html):月历忙碌日 + 今日排期 + 工时打卡
@Composable
fun ScheduleScreen(nav: NavHost) {
    val clockInOkLabel = stringResource(R.string.sched_clock_in_ok)
    val clockOutOkLabel = stringResource(R.string.sched_clock_out_ok)
    var clockHint by remember { mutableStateOf("") }
    var clocking by remember { mutableStateOf(false) }
    val state by loadOnce { ProfileApi.schedule() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val titleLabel = stringResource(R.string.sched_title)

    fun clock(type: String) {
        if (clocking) return
        clocking = true
        scope.launch {
            clockHint = try {
                val r = ProfileApi.clock(type)
                ctx.getString(R.string.sched_clock_ok_fmt,
                    if (type == "IN") clockInOkLabel else clockOutOkLabel,
                    r.optString("clockedAt"))
            } catch (e: Exception) { ctx.getString(R.string.sched_clock_fail, e.message ?: "") }
            clocking = false
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        when (val s = state) {
            is Load.Loading -> { TopBar(titleLabel, onBack = { nav.pop() }); Loading() }
            is Load.Fail -> {
                TopBar(titleLabel, onBack = { nav.pop() })
                Card(Modifier.padding(14.dp)) { Notice(ctx.getString(R.string.sched_load_fail, s.message ?: ""), red = true) }
            }
            is Load.Ok -> {
                val d = s.data
                TopBar(titleLabel, onBack = { nav.pop() }, action = d.optString("month", "-"))
                CalendarCard(d.optString("month", "-"), d.optJSONArray("busyDays"))
                Card(Modifier.padding(12.dp)) {
                    val today = d.optJSONArray("today") ?: JSONArray()
                    SectionTitle(stringResource(R.string.sched_section_today), more = ctx.getString(R.string.sched_today_count, today.length()))
                    if (today.length() == 0) Empty(stringResource(R.string.sched_today_empty))
                    for (i in 0 until today.length()) {
                        val it0 = today.optJSONObject(i)
                        val no = it0.optString("ticketNo")
                        Cell(title = it0.optString("time"),
                            desc = if (no.isEmpty()) it0.optString("address") else "$no · ${it0.optString("address")}",
                            onClick = if (no.isEmpty()) null else ({ nav.push(ticketScreen(no)) }))
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.sched_section_clock))
                    PrimaryButton(stringResource(R.string.sched_btn_in), enabled = !clocking, modifier = Modifier.fillMaxWidth()) { clock("IN") }
                    Spacer(Modifier.height(8.dp))
                    PrimaryButton(stringResource(R.string.sched_btn_out), enabled = !clocking, modifier = Modifier.fillMaxWidth()) { clock("OUT") }
                    if (clockHint.isNotEmpty()) Notice(clockHint)
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun CalendarCard(month: String, busyDays: JSONArray?) {
    val busy = buildSet { for (i in 0 until (busyDays?.length() ?: 0)) add(busyDays?.optInt(i) ?: 0) }
    val year = month.substringBefore("-").toIntOrNull() ?: java.time.LocalDate.now().year
    val monthValue = month.substringAfter("-", "").toIntOrNull() ?: java.time.LocalDate.now().monthValue
    val yearMonth = java.time.YearMonth.of(year, monthValue)
    val firstOffset = yearMonth.atDay(1).dayOfWeek.value - 1
    val totalCells = firstOffset + yearMonth.lengthOfMonth()
    val weekdays = listOf(
        stringResource(R.string.sched_weekday_mon),
        stringResource(R.string.sched_weekday_tue),
        stringResource(R.string.sched_weekday_wed),
        stringResource(R.string.sched_weekday_thu),
        stringResource(R.string.sched_weekday_fri),
        stringResource(R.string.sched_weekday_sat),
        stringResource(R.string.sched_weekday_sun),
    )
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 6.dp)) {
            weekdays.forEach {
                Text(it, fontSize = 12.sp, color = Muted, textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                    modifier = Modifier.weight(1f))
            }
        }
        var cell = 0
        while (cell < totalCells) {
            Row(Modifier.fillMaxWidth()) {
                repeat(7) {
                    Box(Modifier.weight(1f).padding(3.dp), contentAlignment = Alignment.Center) {
                        val day = cell - firstOffset + 1
                        if (cell >= firstOffset && day <= yearMonth.lengthOfMonth()) {
                            val mark = day in busy
                            Text("$day", fontSize = 12.sp,
                                color = if (mark) Color.White else Ink,
                                fontWeight = if (mark) FontWeight.Bold else FontWeight.Normal,
                                modifier = Modifier.size(28.dp).let { m ->
                                    if (mark) m.background(Primary, CircleShape) else m
                                },
                                textAlign = androidx.compose.ui.text.style.TextAlign.Center)
                        }
                        cell++
                    }
                }
            }
        }
    }
}