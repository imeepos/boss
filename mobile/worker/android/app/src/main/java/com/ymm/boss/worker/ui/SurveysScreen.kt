package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.SurveyApi
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.launch
import org.json.JSONArray

// 勘测任务列表(对齐 docs/worker 模式;W7):指派给自己的+未指派抢单池。
@Composable
fun SurveysScreen(nav: NavHost) {
    var refresh by remember { mutableStateOf(0) }
    val state by loadOnce(refresh) { SurveyApi.list() }
    val ctx = LocalContext.current
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.sv_title), onBack = { nav.pop() }, action = stringResource(R.string.tool_action_refresh), onAction = { refresh++ })
        Card(Modifier.padding(12.dp)) {
            when (val s = state) {
                is Load.Loading -> { SectionTitle(stringResource(R.string.sv_section_list)); Loading() }
                is Load.Fail -> { SectionTitle(stringResource(R.string.sv_section_list)); Notice(stringResource(R.string.sv_load_fail), red = true) }
                is Load.Ok -> {
                    val items = s.data.optJSONArray("items") ?: JSONArray()
                    SectionTitle(stringResource(R.string.sv_section_list) + " (" + items.length() + ")")
                    if (items.length() == 0) Empty(stringResource(R.string.sv_empty))
                    for (i in 0 until items.length()) {
                        val t = items.optJSONObject(i) ?: continue
                        val id = t.optLong("id")
                        val status = t.optString("status")
                        val accepted = status == "ACCEPTED" || status == "BACKFILLED"
                        Cell(
                            title = t.optString("taskNo") + " · " + t.optString("title"),
                            desc = svStatusText(status) + " | " +
                                if (t.optLong("assignedWorkerId") > 0) t.optString("workerName") else ctx.getString(R.string.sv_pool) +
                                " | " + ctx.getString(R.string.sv_reports) + " " + t.optInt("reportCount"),
                            onClick = { nav.push(Screen.SurveyDetail(id)) },
                        ) {
                            Text(svStatusText(status), fontSize = 13.sp,
                                color = if (accepted) Success else Primary,
                                modifier = Modifier
                                    .clip(RoundedCornerShape(8.dp))
                                    .clickable { nav.push(Screen.SurveyDetail(id)) }
                                    .padding(horizontal = 8.dp, vertical = 6.dp))
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

// 状态文案映射(与 string 资源解耦,列表轻量展示)。
fun svStatusText(status: String): String = when (status) {
    "ACCEPTED" -> "已接单"
    "BACKFILLED" -> "已回填"
    "CANCELLED" -> "已取消"
    else -> "待执行"
}