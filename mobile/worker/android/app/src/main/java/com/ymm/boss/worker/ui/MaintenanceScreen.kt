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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.ui.theme.TagOrange
import com.ymm.boss.worker.ui.theme.TagRed
import org.json.JSONArray

// 维护清单(对齐 docs/worker/maintenance.html):待替换设备优先级
@Composable
fun MaintenanceScreen(nav: NavHost) {
    val state by loadOnce { AssetApi.maintenance() }
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.maint_title), onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> { SectionTitle(stringResource(R.string.maint_loading_label)); Loading() }
                is Load.Fail -> { SectionTitle(stringResource(R.string.maint_section_priority)); Notice(ctx.getString(R.string.maint_load_fail, s.message ?: ""), red = true) }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    SectionTitle(stringResource(R.string.maint_section_priority), more = ctx.getString(R.string.maint_count_fmt, items.length()))
                    if (items.length() == 0) Empty(stringResource(R.string.maint_empty))
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val parts = buildList {
                            add(ctx.getString(R.string.maint_health_fmt, it0.optString("healthScore")))
                            if (it0.has("faultCount")) add(ctx.getString(R.string.maint_fault_fmt, it0.optInt("faultCount")))
                            if (it0.has("ageYears")) add(ctx.getString(R.string.maint_age_fmt, it0.optString("ageYears")))
                            if (it0.optString("reason").isNotEmpty()) add(it0.optString("reason"))
                        }
                        val urgent = it0.optString("priority") == "MUST_REPLACE"
                        Cell(
                            title = "${it0.optString("deviceNo")} · ${it0.optString("deviceType")}",
                            desc = parts.joinToString(" · "),
                            right = {
                                StatusTag(
                                    it0.optString("priorityLabel"),
                                    color = if (urgent) TagRed else TagOrange,
                                )
                            },
                        )
                    }
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.maint_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}