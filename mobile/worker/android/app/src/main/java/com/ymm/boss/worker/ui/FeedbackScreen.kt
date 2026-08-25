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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Err
import com.ymm.boss.worker.ui.theme.Success
import org.json.JSONArray
import org.json.JSONObject

// 满意度反馈(对齐 docs/worker/feedback.html):最近回访 + 客户反馈
@Composable
fun FeedbackScreen(nav: NavHost) {
    val state by loadOnce { ProfileApi.feedbacks() }
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.fb_title), onBack = { nav.pop() })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(ctx.getString(R.string.fb_load_fail, s.message ?: ""), red = true) }
            is Load.Ok -> {
                val d = s.data
                val latest = d.optJSONObject("latest") ?: JSONObject()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.fb_section_recent))
                    KvRow(latest.optString("ticketNo", "-"), ctx.getString(R.string.fb_score_fmt, latest.optDouble("score")), valueColor = Success)
                    KvRow(stringResource(R.string.fb_section_avg), ctx.getString(R.string.fb_score_fmt, d.optDouble("monthAvgScore")), valueColor = Success)
                    KvRow(stringResource(R.string.fb_recovery), "${d.optDouble("replyRate")}%")
                }
                val items = d.optJSONArray("items") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.fb_kv_customer), more = stringResource(R.string.fb_low_review_note))
                    if (items.length() == 0) Empty(stringResource(R.string.fb_empty))
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val needReview = it0.optBoolean("needReview")
                        val comment = it0.optString("comment")
                        KvRow(
                            it0.optString("customerName"),
                            if (comment.isEmpty()) ctx.getString(R.string.fb_score_fmt, it0.optDouble("score"))
                            else ctx.getString(R.string.fb_score_with_label_fmt, it0.optDouble("score"), comment),
                            valueColor = if (needReview) Err else Success,
                        )
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    Notice(stringResource(R.string.fb_low_review_notice))
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}