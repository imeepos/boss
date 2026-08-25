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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.TicketApi
import org.json.JSONArray

private data class PeriodDef(val code: String, val nameRes: Int)

private val PERIODS = listOf(
    PeriodDef("month", R.string.hist_period_this_month),
    PeriodDef("last-month", R.string.hist_period_last_month),
    PeriodDef("all", R.string.hist_period_all),
)

private fun historyStatus(ctx: android.content.Context, status: String): String = when (status) {
    "DONE" -> ctx.getString(R.string.hist_status_done)
    "CANCELLED", "CANCELED" -> ctx.getString(R.string.hist_status_cancelled)
    else -> status.ifEmpty { ctx.getString(R.string.hist_status_done) }
}

// 历史工单(对齐 docs/worker/history.html):本月/上月/全部 分段时间线
@Composable
fun HistoryScreen(nav: NavHost) {
    val ctx = LocalContext.current
    var cur by remember { mutableStateOf("month") }
    val state by loadOnce(cur) { TicketApi.history(cur) }
    val typeNewInstall = stringResource(R.string.hist_type_new_install)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.hist_title), onBack = { nav.pop() })
        SegRow(cur, { cur = it }, PERIODS.map { it.code to stringResource(it.nameRes) })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> {
                    val name = stringResource(PERIODS.first { it.code == cur }.nameRes)
                    SectionTitle(ctx.getString(R.string.hist_loading, name)); Loading()
                }
                is Load.Fail -> {
                    val name = stringResource(PERIODS.first { it.code == cur }.nameRes)
                    SectionTitle(name); Notice(stringResource(R.string.hist_load_fail), red = true)
                }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    val count = if (items.length() > 0) items.length() else s.data.optInt("totalCount")
                    val name = stringResource(PERIODS.first { it.code == cur }.nameRes)
                    SectionTitle(ctx.getString(R.string.hist_count_fmt, name, count))
                    if (items.length() == 0) Empty(stringResource(R.string.hist_empty))
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val statusLabel = it0.optString("statusLabel").ifEmpty { historyStatus(ctx, it0.optString("status")) }
                        val typeLabel = it0.optString("typeLabel").ifEmpty { typeNewInstall }
                        Cell(
                            title = "${it0.optString("ticketNo")} · $typeLabel",
                            desc = listOfNotNull(
                                it0.optString("address").takeIf { it.isNotEmpty() },
                                it0.optString("finishedAt").takeIf { it.isNotEmpty() },
                            ).joinToString(" · "),
                            onClick = { nav.push(ticketScreen(it0.optString("ticketNo"))) },
                            right = {
                                StatusTag(statusLabel, it0.optString("status"))
                            },
                        )
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}