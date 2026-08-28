package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CardGiftcard
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.PointsApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import com.ymm.boss.user.ui.rememberSubmitGuard
import kotlinx.coroutines.launch
import org.json.JSONObject

// 我的积分页,契约 api/openapi/user/loy.yaml:余额/等级概览 + 赚积分任务 + 积分明细 + 兑换占位。
// 任务完成周期内幂等;兑换因后端暂无模板列表端点(契约要求 templateId)仅占位入口。

@Composable
fun PointsScreen(nav: Nav) {
    var overview by remember { mutableStateOf<JSONObject?>(null) }
    var tierName by remember { mutableStateOf("") }
    var tasks by remember { mutableStateOf(emptyList<JSONObject>()) }
    var offers by remember { mutableStateOf(emptyList<JSONObject>()) }
    var loadErr by remember { mutableStateOf("") }
    var notice by remember { mutableStateOf("") }
    var exchangeErr by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val guard = rememberSubmitGuard()
    var confirmOffer by remember { mutableStateOf<JSONObject?>(null) }

    suspend fun load() {
        loadErr = ""
        try { overview = PointsApi.overview() } catch (e: Exception) { loadErr = "积分加载失败，请检查网络" }
        try { tierName = PointsApi.tier().optJSONObject("tier")?.optString("name").orEmpty() } catch (e: Exception) {}
        try { tasks = PointsApi.tasks().toObjectList() } catch (e: Exception) {}
        try { offers = PointsApi.exchangeOffers().toObjectList() } catch (e: Exception) {}
    }
    LaunchedEffect(nav.refreshTick) { load() }

    fun doExchange(templateId: Long) {
        if (!guard.acquire()) return
        exchangeErr = ""
        notice = ""
        scope.launch {
            try {
                val r = PointsApi.exchange(templateId)
                notice = "兑换成功，优惠券已放入券仓"
                load() // 刷新余额/流水
            } catch (e: Exception) {
                exchangeErr = Api.friendlyMessage(e).ifBlank { "兑换失败，请稍后重试" }
            } finally { guard.release() }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("我的积分", onBack = { nav.pop() })
        if (loadErr.isNotEmpty()) {
            AppCard {
                Text(loadErr, fontSize = 13.sp, color = Palette.err)
                TextButton(onClick = { scope.launch { load() } }) { Text("重试") }
            }
        }
        BalanceCard(overview, tierName)
        ExchangeCard(
            offers = offers,
            balance = overview?.optLong("balance", 0) ?: 0L,
            notice = notice,
            err = exchangeErr,
            onExchange = { o -> confirmOffer = offers.firstOrNull { it.optLong("templateId") == o } },
        )
        TaskCard(tasks, onDone = { scope.launch { load() } })
        EntriesCard(overview)
        Spacer(Modifier.height(12.dp))
    }
    confirmOffer?.let { o ->
        androidx.compose.material3.AlertDialog(
            onDismissRequest = { confirmOffer = null },
            title = { Text("确认兑换") },
            text = { Text("将消耗 ${o.optLong("pointsPrice")} 积分兑换「${o.optString("name")}」？") },
            confirmButton = {
                TextButton(onClick = {
                    confirmOffer = null
                    doExchange(o.optLong("templateId"))
                }) { Text("确认兑换", color = Palette.primary) }
            },
            dismissButton = { TextButton(onClick = { confirmOffer = null }) { Text("取消") } },
        )
    }
}

/** 余额 + 等级概览卡:金额主色高亮,与账单/支付页同一视觉语言。 */
@Composable
private fun BalanceCard(overview: JSONObject?, tierName: String) {
    val balance = overview?.optLong("balance", 0) ?: 0L
    AppCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("积分余额", fontSize = 14.sp, color = Palette.muted)
                Row(verticalAlignment = Alignment.Bottom) {
                    Text("$balance", fontSize = 30.sp, fontWeight = FontWeight.Bold, color = Palette.primary)
                    Text(" 积分", fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(bottom = 4.dp))
                }
            }
            if (tierName.isNotBlank()) Tag(tierName, Palette.orange)
        }
        Spacer(Modifier.height(8.dp))
        Text(
            "缴费和完成任务可获得积分，积分可兑换优惠券，过期自动清理。",
            fontSize = 12.sp, color = Palette.muted,
        )
    }
}

/**
 * 积分兑换卡:可兑换券模板列表(契约 /points/exchange-offers)。
 * offers 为空时保持占位终态(暂无兑换券),不渲染假功能;
 * 兑换走 /points/exchange(先扣后发,后端补偿回补),成功余额刷新 + 券到账终态文案。
 */
@Composable
private fun ExchangeCard(
    offers: List<JSONObject>,
    balance: Long,
    notice: String,
    err: String,
    onExchange: (Long) -> Unit,
) {
    AppCard {
        CardTitle("积分兑换")
        if (offers.isEmpty()) {
            EmptyState("暂无可兑换券")
        } else {
            offers.forEachIndexed { i, o ->
                val price = o.optLong("pointsPrice")
                val affordable = balance >= price
                Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    IconTile(Icons.Filled.CardGiftcard, Palette.orange, size = 40.dp, corner = 12.dp)
                    Spacer(Modifier.width(12.dp))
                    Column(Modifier.weight(1f)) {
                        Text(o.optString("name"), fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink, maxLines = 1)
                        Spacer(Modifier.height(3.dp))
                        Text(offerDesc(o) + " · $price 积分", fontSize = 11.5.sp, color = Palette.muted)
                    }
                    Spacer(Modifier.width(8.dp))
                    if (affordable) {
                        Button(
                            onClick = { onExchange(o.optLong("templateId")) },
                            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                            contentPadding = PaddingValues(horizontal = 14.dp, vertical = 6.dp),
                            modifier = Modifier.height(34.dp),
                        ) { Text("兑换", fontSize = 13.sp) }
                    } else {
                        Tag("积分不足", Palette.subtle)
                    }
                }
                if (i < offers.lastIndex) HorizontalDivider(color = Palette.line, thickness = 0.5.dp)
            }
        }
        if (notice.isNotBlank()) Text(notice, fontSize = 12.5.sp, color = Palette.success, modifier = Modifier.padding(top = 4.dp))
        if (err.isNotBlank()) Text(err, fontSize = 12.5.sp, color = Palette.err, modifier = Modifier.padding(top = 4.dp))
    }
}

private fun offerDesc(o: JSONObject): String {
    val face = o.optLong("faceValue")
    val threshold = o.optLong("threshold")
    return when (o.optString("type")) {
        "CASH" -> "¥%.2f 无门槛券".format(face / 100.0)
        "FULL_CUT" -> "满 ¥%.2f 减 ¥%.2f".format(threshold / 100.0, face / 100.0)
        else -> "优惠券"
    }
}

/** 赚积分任务列表:未完成可点击领取(防重复提交,领取中转圈),完成后显示已完成。 */
@Composable
private fun TaskCard(tasks: List<JSONObject>, onDone: () -> Unit) {
    val scope = rememberCoroutineScope()
    var submittingId by remember { mutableStateOf<Long?>(null) }
    var submitErr by remember { mutableStateOf("") }
    AppCard {
        CardTitle("赚积分")
        if (tasks.isEmpty()) {
            EmptyState("暂无积分任务")
        } else {
            tasks.forEachIndexed { i, t ->
                val taskId = t.optLong("taskId")
                val done = t.optString("completedAt").isNotBlank()
                Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("+${t.optLong("points")}", fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Palette.primary)
                            Spacer(Modifier.width(8.dp))
                            Text(t.optString("name"), fontSize = 14.sp, color = Palette.ink, maxLines = 1)
                        }
                        Spacer(Modifier.height(4.dp))
                        Text(periodLabel(t.optString("period")), fontSize = 11.5.sp, color = Palette.subtle)
                    }
                    Spacer(Modifier.width(8.dp))
                    if (done) {
                        Tag("已完成", Palette.success)
                    } else {
                        val submitting = submittingId == taskId && taskId > 0
                        Button(
                            onClick = {
                                if (submittingId != null) return@Button
                                submittingId = taskId
                                submitErr = ""
                                scope.launch {
                                    try {
                                        PointsApi.completeTask(taskId)
                                        onDone()
                                    } catch (e: Exception) {
                                        submitErr = "领取失败，请重试"
                                    } finally {
                                        submittingId = null
                                    }
                                }
                            },
                            enabled = !submitting,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = Palette.primary,
                                disabledContainerColor = Palette.primary.copy(alpha = 0.4f),
                            ),
                            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 6.dp),
                            modifier = Modifier.height(34.dp),
                        ) { Text(if (submitting) "领取中…" else "去完成", fontSize = 13.sp) }
                    }
                }
                if (i < tasks.lastIndex) HorizontalDivider(color = Palette.line, thickness = 0.5.dp)
            }
        }
        if (submitErr.isNotEmpty()) Text(submitErr, fontSize = 12.5.sp, color = Palette.err)
    }
}

/** 积分流水明细:获得绿色加号、消耗红色减号,原因按 terms.md 枚举映射。 */
@Composable
private fun EntriesCard(overview: JSONObject?) {
    val entries = overview?.optJSONArray("entries").toObjectList()
    AppCard {
        CardTitle("积分明细")
        if (entries.isEmpty()) {
            EmptyState("暂无积分记录")
        } else {
            entries.forEachIndexed { i, e ->
                val delta = e.optLong("delta")
                Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text(reasonLabel(e.optString("reason")), fontSize = 14.sp, color = Palette.ink, maxLines = 1)
                        Spacer(Modifier.height(3.dp))
                        Text(e.optString("createdAt"), fontSize = 11.5.sp, color = Palette.subtle)
                    }
                    Row(verticalAlignment = Alignment.Bottom) {
                        Text(
                            if (delta >= 0) "+$delta" else "$delta",
                            fontSize = 15.sp, fontWeight = FontWeight.Bold,
                            color = if (delta >= 0) Palette.success else Palette.err,
                        )
                        Text(" 积分", fontSize = 11.sp, color = Palette.subtle, modifier = Modifier.padding(bottom = 1.dp))
                    }
                }
                if (i < entries.lastIndex) HorizontalDivider(color = Palette.line, thickness = 0.5.dp)
            }
        }
    }
}

private fun periodLabel(p: String): String = when (p) {
    "DAILY" -> "每日任务"
    "MONTHLY" -> "每月任务"
    "ONE_TIME" -> "一次性任务"
    else -> p
}

private fun reasonLabel(r: String): String = when (r) {
    "ADMIN_ADJUST" -> "手动调整"
    "EXCHANGE" -> "积分兑换"
    "EXCHANGE_REVERSAL" -> "兑换回补"
    "PAYMENT_EARN" -> "缴费奖励"
    "PAYMENT_REVERSAL" -> "缴费回退"
    "TASK_EARN" -> "任务奖励"
    "EXPIRED" -> "过期清理"
    "COMPENSATION" -> "补偿"
    else -> r
}