package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.ui.theme.TagOrange
import com.ymm.boss.worker.ui.theme.TagRed
import org.json.JSONArray

// 维护清单(对齐 docs/worker/maintenance.html):待替换设备优先级
@Composable
fun MaintenanceScreen(nav: NavHost) {
    val state by loadOnce { AssetApi.maintenance() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("维护清单", onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> { SectionTitle("待替换设备优先级 · 加载中…"); Loading() }
                is Load.Fail -> { SectionTitle("待替换设备优先级"); Notice("加载失败：${s.message}", red = true) }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    SectionTitle("待替换设备优先级", more = "${items.length()} 台")
                    if (items.length() == 0) Empty("暂无待替换设备")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val parts = buildList {
                            add("健康度 ${it0.optString("healthScore")}")
                            if (it0.has("faultCount")) add("故障 ${it0.optInt("faultCount")} 次")
                            if (it0.has("ageYears")) add("在网 ${it0.optString("ageYears")} 年")
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
            Notice("清单按健康度/故障率/使用年限综合排序，评分规则公开可解释。")
        }
        Spacer(Modifier.height(12.dp))
    }
}