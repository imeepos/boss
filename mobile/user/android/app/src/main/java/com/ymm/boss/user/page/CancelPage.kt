package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
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
import com.ymm.boss.user.api.PlanApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 对应草稿 docs/user/cancel.html:退订拆机,先 GET 预览待结清款项再确认提交。
private val REASONS = listOf(
    "leave_area" to "迁离服务区",
    "switch_operator" to "更换运营商",
    "cost" to "费用过高",
    "other" to "其他",
)

@Composable
fun CancelScreen(nav: Nav, planId: String) {
    var unpaid by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var penalty by remember { mutableStateOf("—") }
    var penaltyDesc by remember { mutableStateOf("") }
    var reason by remember { mutableStateOf("leave_area") }
    var err by remember { mutableStateOf("") }
    var done by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(nav.refreshTick) {
        try { loadCancelPreview(planId, { unpaid = it.first; penalty = it.second; penaltyDesc = it.third }) }
        catch (e: Exception) { err = "退订预检加载失败,可稍后重试" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("退订拆机", onBack = { nav.pop() })
        if (err.isNotBlank()) Notice(err, Palette.err)
        if (done.isNotBlank()) { DoneCard(done, onBack = { nav.pop() }) } else {
            NoticeCard()
            UnpaidCard(unpaid, penalty, penaltyDesc)
            ReasonCard(reason, onPick = { reason = it })
            SubmitBar("确认退订 · 生成拆机单", onSubmit = {
                submitCancel(scope, planId, reason,
                    onDone = { done = it }, onErr = { err = it })
            })
        }
        Spacer(Modifier.height(16.dp))
    }
}

private suspend fun loadCancelPreview(planId: String, set: (Triple<List<JSONObject>, String, String>) -> Unit) {
    val d = PlanApi.cancelPreview(planId)
    val bills = d.optJSONArray("unpaidBills") ?: JSONArray()
    val unpaid = (0 until bills.length()).mapNotNull { bills.optJSONObject(it) }
    set(Triple(unpaid, "¥%.2f".format(d.optDouble("penalty")), d.optString("penaltyDesc")))
}

private fun submitCancel(
    scope: kotlinx.coroutines.CoroutineScope, planId: String, reason: String,
    onDone: (String) -> Unit, onErr: (String) -> Unit,
) {
    scope.launch {
        try {
            val o = PlanApi.cancel(planId, reason)
            onDone("退订申请已提交,拆机工单号 ${o.optString("orderNo")}")
        } catch (e: Exception) { onErr("提交失败,请稍后重试") }
    }
}

@Composable
private fun NoticeCard() {
    AppCard {
        CardTitle("退订须知")
        val lines = listOf(
            "1. 合约期内退订,按未履约月份收取违约金,具体以结算页为准。",
            "2. 拆机须装维师傅上门扫码解绑光猫设备(强制节点),请保持联系畅通。",
            "3. 拆机完成后端口与账号同步释放,未缴账单需结清后方可完成。",
        )
        lines.forEach {
            Text(it, fontSize = 13.sp, color = Palette.muted, modifier = Modifier.padding(vertical = 3.dp))
        }
    }
}

@Composable
private fun UnpaidCard(unpaid: List<JSONObject>, penalty: String, penaltyDesc: String) {
    AppCard {
        CardTitle("待结清款项")
        if (unpaid.isEmpty()) EmptyState("暂无未缴账单")
        unpaid.forEach { b ->
            CellRow(
                title = "${b.optString("period")} 账期",
                onClick = null,
                right = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("¥%.2f".format(b.optDouble("amount")), fontSize = 13.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
                        Spacer(Modifier.width(8.dp))
                        Tag("未缴", Palette.orange)
                    }
                },
            )
        }
        CellRow("合约违约金", penaltyDesc.ifBlank { "以结算页为准" }, right = {
            Text(penalty, fontSize = 13.sp, fontWeight = FontWeight.W600, color = Palette.err)
        })
    }
}

@Composable
private fun ReasonCard(reason: String, onPick: (String) -> Unit) {
    AppCard {
        CardTitle("退订原因")
        REASONS.forEach { (value, label) ->
            Row(
                Modifier.fillMaxWidth().clickable { onPick(value) }.padding(vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    label,
                    fontSize = 14.sp,
                    fontWeight = if (reason == value) FontWeight.W600 else FontWeight.Normal,
                    color = if (reason == value) Palette.primary else Palette.ink,
                    modifier = Modifier.weight(1f),
                )
                if (reason == value) Tag("已选", Palette.primary)
            }
        }
    }
}
