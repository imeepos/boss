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
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Err
import com.ymm.boss.worker.ui.theme.Success
import org.json.JSONArray
import org.json.JSONObject

// 满意度反馈(对齐 docs/worker/feedback.html):最近回访 + 客户反馈
@Composable
fun FeedbackScreen(nav: NavHost) {
    val state by loadOnce { ProfileApi.feedbacks() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("满意度反馈", onBack = { nav.pop() })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("反馈加载失败：${s.message}", red = true) }
            is Load.Ok -> {
                val d = s.data
                val latest = d.optJSONObject("latest") ?: JSONObject()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle("最近回访")
                    KvRow(latest.optString("ticketNo", "-"), "${latest.optDouble("score")} 分", valueColor = Success)
                    KvRow("本月平均", "${d.optDouble("monthAvgScore")} 分", valueColor = Success)
                    KvRow("评价回收率", "${d.optDouble("replyRate")}%")
                }
                val items = d.optJSONArray("items") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle("客户反馈", more = "低分需复核")
                    if (items.length() == 0) Empty("暂无反馈")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val needReview = it0.optBoolean("needReview")
                        KvRow(
                            it0.optString("customerName"),
                            "${it0.optDouble("score")} 分${it0.optString("comment").let { if (it.isEmpty()) "" else " · $it" }}",
                            valueColor = if (needReview) Err else Success,
                        )
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    Notice("满意度 < 3 分自动升级主管复核，避免争议。")
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}