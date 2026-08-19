package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.TicketApi
import org.json.JSONArray

private val PERIODS = listOf("month" to "本月", "last-month" to "上月", "all" to "全部")
private val PERIOD_NAMES = mapOf("month" to "本月", "last-month" to "上月", "all" to "全部")

// 历史工单(对齐 docs/worker/history.html):本月/上月/全部 分段时间线
@Composable
fun HistoryScreen(nav: NavHost) {
    var cur by remember { mutableStateOf("month") }
    val state by loadOnce(cur) { TicketApi.history(cur) }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("历史工单", onBack = { nav.pop() })
        SegRow(cur, { cur = it }, PERIODS)
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> { SectionTitle("${PERIOD_NAMES.getValue(cur)} · 加载中…"); Loading() }
                is Load.Fail -> { SectionTitle(PERIOD_NAMES.getValue(cur)); Notice("历史工单加载失败，请重试。", red = true) }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    val count = if (items.length() > 0) items.length() else s.data.optInt("totalCount")
                    SectionTitle("${PERIOD_NAMES[cur]} · $count 单")
                    if (items.length() == 0) Empty("暂无历史工单")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        Cell(
                            title = "${it0.optString("ticketNo")} · ${it0.optString("typeLabel")}",
                            desc = listOfNotNull(
                                it0.optString("address").takeIf { it.isNotEmpty() },
                                it0.optString("finishedAt").takeIf { it.isNotEmpty() },
                            ).joinToString(" · "),
                            onClick = { nav.push(ticketScreen(it0.optString("ticketNo"))) },
                        ) { StatusTag(it0.optString("statusLabel"), it0.optString("status")) }
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}