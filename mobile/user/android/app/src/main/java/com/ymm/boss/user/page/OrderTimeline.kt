package com.ymm.boss.user.page

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Autorenew
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.HourglassEmpty
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.Palette
import org.json.JSONObject

// 12 环节装维时间线:进度条 + 4 里程碑分组折叠卡。纯逻辑抽出为可测顶层函数。

/** 当前里程碑 = timeline 最大 stage 所在分组,1~4。 */
internal fun currentMilestoneOf(stages: List<Int>): Int =
    (((stages.maxOrNull() ?: 1) - 1) / 3 + 1).coerceIn(1, 4)

/** 里程碑状态:依据该里程碑下各阶段 result 推导。 */
internal enum class MilestoneState { DONE, DOING, PENDING }

internal fun milestoneStateOf(results: List<String>): MilestoneState = when {
    results.isEmpty() -> MilestoneState.PENDING
    results.all { it == "DONE" } -> MilestoneState.DONE
    results.any { it == "DOING" || it == "DONE" } -> MilestoneState.DOING
    else -> MilestoneState.PENDING
}

@Composable
internal fun OrderTimelineCard(timeline: List<JSONObject>) {
    AppCard {
        val done = timeline.count { it.optString("result") == "DONE" }
        val currentMilestone = currentMilestoneOf(timeline.map { it.optInt("stage", 1) })
        val allDone = timeline.isNotEmpty() && done == timeline.size
        CardTitle("装维进度", "$done/12 已完成")
        Spacer(Modifier.height(8.dp))
        ProgressBar(done, 12)
        Spacer(Modifier.height(4.dp))
        Text(
            "共 12 个步骤,已完成 $done 个",
            fontSize = 11.5.sp, color = Palette.muted,
        )
        Spacer(Modifier.height(12.dp))
        if (timeline.isEmpty()) {
            Text("进度加载中…", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 4.dp))
            return@AppCard
        }
        val byMilestone = (1..4).associateWith { m ->
            timeline.filter { currentMilestoneOf(listOf(it.optInt("stage", 1))) == m }
        }
        byMilestone.forEach { (m, stages) ->
            if (stages.isNotEmpty()) {
                MilestoneGroup(m, stages, currentMilestone, allDone)
                Spacer(Modifier.height(8.dp))
            }
        }
    }
}

@Composable
private fun ProgressBar(done: Int, total: Int) {
    val fraction = if (total <= 0) 0f else (done.toFloat() / total).coerceIn(0f, 1f)
    Box(
        Modifier.fillMaxWidth().height(4.dp).background(Palette.line, RoundedCornerShape(2.dp)),
    ) {
        Box(
            Modifier.fillMaxHeight().fillMaxWidth(fraction).background(Palette.primary, RoundedCornerShape(2.dp)),
        )
    }
}

@Composable
private fun MilestoneGroup(
    milestone: Int,
    stages: List<JSONObject>,
    currentMilestone: Int,
    allDone: Boolean,
) {
    val name = when (milestone) {
        1 -> "提交订单"; 2 -> "受理成功"; 3 -> "上门安装"; 4 -> "完成"
        else -> "—"
    }
    val state = milestoneStateOf(stages.map { it.optString("result") })
    val doneIn = stages.count { it.optString("result") == "DONE" }
    val totalIn = stages.size
    // 默认展开:当前里程碑 / 全部已完成 / 过去里程碑保持折叠
    val defaultExpanded = allDone || state == MilestoneState.DOING
    var expanded by remember { mutableStateOf(defaultExpanded) }

    val isCurrent = state == MilestoneState.DOING && !allDone
    val bg = if (isCurrent) Palette.primary.copy(alpha = 0.06f) else Palette.panel
    val border = if (isCurrent) BorderStroke(1.dp, Palette.primary.copy(alpha = 0.2f)) else null

    Column(
        Modifier.fillMaxWidth()
            .then(if (border != null) Modifier.border(border, RoundedCornerShape(12.dp)) else Modifier)
            .background(bg, RoundedCornerShape(12.dp))
            .padding(horizontal = 14.dp, vertical = 12.dp),
    ) {
        Row(
            Modifier.fillMaxWidth().clickable { expanded = !expanded },
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MilestoneBadge(state)
            Spacer(Modifier.width(10.dp))
            Text(
                name,
                fontSize = 16.sp, fontWeight = FontWeight.W600,
                color = if (state == MilestoneState.PENDING) Palette.muted else Palette.ink,
                modifier = Modifier.weight(1f),
            )
            Text("$doneIn/$totalIn", fontSize = 13.sp, color = Palette.muted)
            Spacer(Modifier.width(8.dp))
            Chevron(expanded)
        }
        if (expanded) {
            Spacer(Modifier.height(8.dp))
            stages.forEachIndexed { i, t -> StageRow(t, isLast = i == stages.size - 1) }
        }
    }
}

@Composable
private fun MilestoneBadge(state: MilestoneState) {
    val (bg, fg, icon) = when (state) {
        MilestoneState.DONE -> Triple(Palette.success.copy(alpha = 0.12f), Palette.success, Icons.Filled.Check)
        MilestoneState.DOING -> Triple(Palette.primary.copy(alpha = 0.12f), Palette.primary, Icons.Filled.Autorenew)
        MilestoneState.PENDING -> Triple(Palette.muted.copy(alpha = 0.10f), Palette.muted, Icons.Filled.HourglassEmpty)
    }
    Box(
        Modifier.size(28.dp).background(bg, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, contentDescription = null, tint = fg, modifier = Modifier.size(16.dp))
    }
}

@Composable
private fun Chevron(expanded: Boolean) {
    val rotation by animateFloatAsState(
        targetValue = if (expanded) 180f else 0f,
        animationSpec = tween(durationMillis = 150),
        label = "chevron-rotation",
    )
    Icon(
        Icons.AutoMirrored.Filled.KeyboardArrowRight,
        contentDescription = if (expanded) "收起" else "展开",
        tint = Palette.subtle, modifier = Modifier.size(20.dp).graphicsLayer { rotationZ = rotation },
    )
}

@Composable
private fun StageRow(t: JSONObject, isLast: Boolean) {
    val result = t.optString("result")
    Row(Modifier.height(IntrinsicSize.Min).padding(vertical = 4.dp)) {
        Column(Modifier.width(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            StageNode(result)
            if (!isLast) Box(Modifier.width(2.dp).fillMaxHeight().background(Palette.line))
        }
        Column(Modifier.padding(start = 10.dp, bottom = if (isLast) 0.dp else 8.dp).weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    "${t.optInt("stage", 1)} ${t.optString("title")}",
                    fontSize = 14.sp,
                    color = when (result) {
                        "PENDING" -> Palette.muted
                        "DOING" -> Palette.primary
                        else -> Palette.ink
                    },
                    fontWeight = if (result == "DOING") FontWeight.Bold else FontWeight.W500,
                    modifier = Modifier.weight(1f),
                )
                val ts = t.optString("finishedAt").orEmpty()
                if (ts.isNotBlank()) Text(ts, fontSize = 12.sp, color = Palette.muted)
            }
            val meta = t.optString("meta").orEmpty()
            if (meta.isNotBlank()) {
                Text(meta, fontSize = 11.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 2.dp))
            }
        }
    }
}

@Composable
private fun StageNode(result: String) {
    when (result) {
        "DONE" -> Box(Modifier.size(12.dp).background(Palette.success, CircleShape))
        "DOING" -> Box(
            Modifier.size(14.dp).background(Palette.primary, CircleShape)
                .border(3.dp, Palette.primary.copy(alpha = 0.25f), CircleShape),
        )
        else -> Box(
            Modifier.size(12.dp).background(Palette.panel, CircleShape).border(1.5.dp, Palette.line, CircleShape),
        )
    }
}
