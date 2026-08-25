package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.Warn
import org.json.JSONArray

// 绩效明细(对齐 docs/worker/performance.html):本月绩效 + 提成 + 小组排名
@Composable
fun PerformanceScreen(nav: NavHost) {
    val state by loadOnce { ProfileApi.performance() }
    val ctx = LocalContext.current
    val title = stringResource(R.string.perf_section_commission)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        when (val s = state) {
            is Load.Loading -> { TopBar(title, onBack = { nav.pop() }); Loading() }
            is Load.Fail -> {
                TopBar(title, onBack = { nav.pop() })
                Card(Modifier.padding(14.dp)) { Notice(ctx.getString(R.string.perf_load_fail, s.message ?: ""), red = true) }
            }
            is Load.Ok -> {
                val d = s.data
                val period = d.optString("period", "-")
                TopBar(title, onBack = { nav.pop() }, action = period)
                val sum = d.optJSONObject("summary") ?: org.json.JSONObject()
                StatCard(
                    stringResource(R.string.perf_title), period,
                    listOf(
                        Triple(sum.optString("finished", "0"), stringResource(R.string.perf_stat_done), Primary),
                        Triple(sum.optString("onTimeRate", "0") + "%", stringResource(R.string.perf_stat_on_time), Success),
                        Triple(sum.optString("score", "0"), stringResource(R.string.perf_kv_score), Warn),
                    ),
                )
                val commissions = d.optJSONArray("commissions") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.perf_section_commission))
                    if (commissions.length() == 0) Empty(stringResource(R.string.perf_empty_commission))
                    for (i in 0 until commissions.length()) {
                        val c = commissions.optJSONObject(i)
                        val formula = c.optString("formula")
                        val amt = c.optDouble("amount")
                        KvRow(c.optString("name"),
                            (if (formula.isNotEmpty()) "$formula = " else "") + "¥$amt")
                    }
                    val total = d.optDouble("totalAmount", 0.0)
                    KvRow(stringResource(R.string.perf_stat_total), "¥$total", valueColor = Primary)
                }
                val ranking = d.optJSONArray("ranking") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.perf_stat_rank))
                    if (ranking.length() == 0) Empty(stringResource(R.string.perf_empty_rank))
                    for (i in 0 until ranking.length()) {
                        val r = ranking.optJSONObject(i)
                        KvRow(r.optString("name"), "#${r.optInt("rank")}")
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}