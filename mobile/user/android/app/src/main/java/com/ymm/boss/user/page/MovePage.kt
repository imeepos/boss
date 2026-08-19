package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.PlanApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/move.html:迁址移机,填新地址后提交生成迁址工单。
// 契约: POST /plans/{planId}/move {community,building,door,expectDate} -> OrderSummary
@Composable
fun MoveScreen(nav: Nav, planId: String) {
    var oldAddr by remember { mutableStateOf("—") }
    var oldPlan by remember { mutableStateOf("—") }
    var community by remember { mutableStateOf("") }
    var building by remember { mutableStateOf("") }
    var door by remember { mutableStateOf("") }
    var expectDate by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    var done by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) { loadOldAddress { oldAddr = it.first; oldPlan = it.second } }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("迁址移机") { nav.pop() }
        if (err.isNotBlank()) Notice(err, Palette.err)
        if (done.isNotBlank()) { DoneCard(done) { nav.pop() } } else {
            MoveForm(oldAddr, oldPlan, community, building, door, expectDate,
                onCommunity = { community = it }, onBuilding = { building = it },
                onDoor = { door = it }, onDate = { expectDate = it }) {
                submitMove(scope, planId, community, building, door, expectDate,
                    onDone = { done = it }, onErr = { err = it })
            }
        }
        Spacer(Modifier.height(16.dp))
    }
}

@Composable
private fun MoveForm(
    oldAddr: String, oldPlan: String,
    community: String, building: String, door: String, expectDate: String,
    onCommunity: (String) -> Unit, onBuilding: (String) -> Unit,
    onDoor: (String) -> Unit, onDate: (String) -> Unit,
    onSubmit: () -> Unit,
) {
    AppCard {
        CardTitle("原安装地址")
        CellRow(oldAddr, oldPlan, right = { Tag("在用", Palette.success) })
    }
    AppCard {
        CardTitle("新安装地址")
        AddressFields(community, building, door, expectDate,
            onCommunity = onCommunity, onBuilding = onBuilding, onDoor = onDoor, onDate = onDate)
        Text(
            "提交后系统将自动做资源核查,如有空闲端口即生成迁址工单并派单。",
            fontSize = 12.sp, color = Palette.muted,
            modifier = Modifier.padding(top = 8.dp),
        )
    }
    SubmitBar("提交迁址申请", enabled = community.isNotBlank() && building.isNotBlank() && door.isNotBlank(), onSubmit = onSubmit)
}

private suspend fun loadOldAddress(set: (Pair<String, String>) -> Unit) {
    try {
        val p = PlanApi.profile().optJSONObject("plan") ?: JSONObject()
        set(p.optString("installAddress").ifBlank { "—" } to "在用 · ${p.optString("name")}")
    } catch (e: Exception) { set("原安装地址" to "—") }
}

private fun submitMove(
    scope: kotlinx.coroutines.CoroutineScope, planId: String,
    community: String, building: String, door: String, expectDate: String,
    onDone: (String) -> Unit, onErr: (String) -> Unit,
) {
    scope.launch {
        try {
            val o = PlanApi.move(planId, community, building, door, expectDate)
            onDone("迁址申请已提交,工单号 ${o.optString("orderNo")}")
        } catch (e: Exception) { onErr("提交失败,请稍后重试") }
    }
}

@Composable
private fun AddressFields(
    community: String, building: String, door: String, expectDate: String,
    onCommunity: (String) -> Unit, onBuilding: (String) -> Unit,
    onDoor: (String) -> Unit, onDate: (String) -> Unit,
) {
    Column(Modifier.padding(top = 6.dp)) {
        FieldLabel("小区 / 楼盘")
        OutlinedTextField(value = community, onValueChange = onCommunity, singleLine = true,
            placeholder = { Text("请输入新小区名") }, modifier = Modifier.fillMaxWidth())
        FieldLabel("楼栋")
        OutlinedTextField(value = building, onValueChange = onBuilding, singleLine = true,
            placeholder = { Text("楼栋号") }, modifier = Modifier.fillMaxWidth())
        FieldLabel("门牌号")
        OutlinedTextField(value = door, onValueChange = onDoor, singleLine = true,
            placeholder = { Text("单元/房间号") }, modifier = Modifier.fillMaxWidth())
        FieldLabel("期望移机时间")
        OutlinedTextField(value = expectDate, onValueChange = onDate, singleLine = true,
            placeholder = { Text("如 2025-09-01") },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth())
    }
}
