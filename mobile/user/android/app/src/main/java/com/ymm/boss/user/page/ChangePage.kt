package com.ymm.boss.user.page

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
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.RadioButton
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
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PricePill
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 对应草稿 docs/user/change.html:改套餐,选新套餐+生效方式+可选静态 IP。
// 契约: POST /plans/{planId}/change {targetPlanId,effectiveMode,staticIp} -> OrderSummary
@Composable
fun ChangeScreen(nav: Nav, planId: String) {
    var curName by remember { mutableStateOf("—") }
    var curDesc by remember { mutableStateOf("—") }
    var options by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var selected by remember { mutableStateOf("") }
    var effMode by remember { mutableStateOf("immediate") }
    var staticIp by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var done by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(nav.refreshTick) { loadCurrent { curName = it.first; curDesc = it.second } }
    LaunchedEffect(nav.refreshTick) { loadOptions({ options = it }, { selected = it }, { err = "套餐列表加载失败" }) }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("改套餐", onBack = { nav.pop() })
        if (err.isNotBlank()) Notice(err, Palette.err)
        if (done.isNotBlank()) { DoneCard(done, onBack = { nav.pop() }) } else {
            ChangeBody(
                curName, curDesc, options, selected, effMode, staticIp,
                onSelect = { selected = it }, onEff = { effMode = it }, onIp = { staticIp = it },
                onSubmit = {
                    submitChange(scope, planId, selected, effMode, staticIp,
                        onDone = { done = it }, onErr = { err = it })
                },
            )
        }
        Spacer(Modifier.height(16.dp))
    }
}

private suspend fun loadCurrent(set: (Pair<String, String>) -> Unit) {
    try {
        val p = PlanApi.profile().optJSONObject("plan") ?: JSONObject()
        set(p.optString("name") to "¥${p.optString("monthlyFee")}/月 · 合约至 ${p.optString("contractEnd")}")
    } catch (e: Exception) { set("当前套餐" to "—") }
}

private suspend fun loadOptions(setList: (List<JSONObject>) -> Unit, setSel: (String) -> Unit, onErr: () -> Unit) {
    try {
        val items = PlanApi.products().optJSONArray("items") ?: JSONArray()
        val list = (0 until items.length()).mapNotNull { items.optJSONObject(it) }
        setList(list)
        val preferred = list.firstOrNull { it.optString("productId") == "P-2000" } ?: list.firstOrNull()
        preferred?.let { setSel(it.optString("productId")) }
    } catch (e: Exception) { onErr() }
}

private fun submitChange(
    scope: kotlinx.coroutines.CoroutineScope, planId: String,
    selected: String, effMode: String, staticIp: Boolean,
    onDone: (String) -> Unit, onErr: (String) -> Unit,
) {
    scope.launch {
        try {
            val o = PlanApi.change(planId, selected, effMode, staticIp)
            onDone("变更申请已提交,工单号 ${o.optString("orderNo")}")
        } catch (e: Exception) { onErr("提交失败,请稍后重试") }
    }
}

@Composable
private fun ChangeBody(
    curName: String, curDesc: String, options: List<JSONObject>,
    selected: String, effMode: String, staticIp: Boolean,
    onSelect: (String) -> Unit, onEff: (String) -> Unit, onIp: (Boolean) -> Unit,
    onSubmit: () -> Unit,
) {
    AppCard {
        CardTitle("当前套餐")
        CellRow(curName, curDesc, right = { Tag("在网", Palette.success) })
    }
    PlanOptionsCard(options, selected, onSelect)
    AppCard {
        CardTitle("生效方式")
        EffOption("即时生效", "当月按新旧价按日折算", "immediate", effMode, onEff)
        EffOption("预约生效", "次月 1 日生效,当月按原价", "scheduled", effMode, onEff)
    }
    StaticIpCard(staticIp, onIp)
    SubmitBar("提交变更申请", enabled = selected.isNotBlank(), onSubmit = onSubmit)
}

@Composable
private fun PlanOptionsCard(options: List<JSONObject>, selected: String, onSelect: (String) -> Unit) {
    AppCard {
        CardTitle("选择新套餐")
        options.forEach { p ->
            val pid = p.optString("productId")
            CellRow(
                title = p.optString("name"),
                desc = p.optString("description"),
                onClick = { onSelect(pid) },
                right = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        PricePill(p.optString("monthlyFee"), recommended = p.optBoolean("featured"))
                        Spacer(Modifier.width(6.dp))
                        RadioButton(selected = selected == pid, onClick = { onSelect(pid) })
                    }
                },
            )
        }
    }
}

@Composable
private fun StaticIpCard(staticIp: Boolean, onIp: (Boolean) -> Unit) {
    AppCard {
        CardTitle("静态 IP 申请 可选")
        Row(Modifier.fillMaxWidth().padding(top = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("公网静态 IPv4", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink)
                Text("¥30/月 · 需审批,1~2 个工作日", fontSize = 12.sp, color = Palette.muted)
            }
            Checkbox(checked = staticIp, onCheckedChange = onIp)
        }
    }
}

@Composable
private fun EffOption(title: String, desc: String, value: String, current: String, onPick: (String) -> Unit) {
    CellRow(title, desc, onClick = { onPick(value) }, right = {
        RadioButton(selected = current == value, onClick = { onPick(value) })
    })
}

@Composable
internal fun SubmitBar(label: String, enabled: Boolean = true, onSubmit: () -> Unit) {
    Button(
        onClick = onSubmit,
        enabled = enabled,
        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp).height(44.dp),
    ) { Text(label) }
}

@Composable
internal fun DoneCard(message: String, onBack: () -> Unit) {
    AppCard {
        Text(message, fontSize = 14.sp, color = Palette.ink)
        Button(
            onClick = onBack,
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp).height(40.dp),
        ) { Text("返回上一页") }
    }
}
