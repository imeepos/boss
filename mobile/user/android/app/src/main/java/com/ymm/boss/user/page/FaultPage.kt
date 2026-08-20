package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.HeadsetMic
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.FaultApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/fault.html:报修表单 + 报修记录列表。
@Composable
fun FaultScreen(nav: Nav) {
    val scope = rememberCoroutineScope()
    val form = remember { FaultFormState() }
    var records by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var msg by remember { mutableStateOf("") }

    fun reload() {
        scope.launch {
            try {
                records = FaultApi.list().optJSONArray("items").toObjList()
            } catch (e: Exception) {
                // 列表不可达时保留空骨架,与草稿一致
            }
        }
    }
    LaunchedEffect(nav.refreshTick) { reload() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("故障报修", onBack = { nav.pop() })
        QuickEntries(nav)
        FaultFormCard(scope, nav, form) { msg = it; reload() }
        Notice(msg, if (msg == "已受理") Palette.success else Palette.err)
        AppCard {
            CardTitle("报修记录")
            if (records.isEmpty()) EmptyState("暂无报修记录")
            records.forEach { f -> FaultCell(f) { nav.push(Route.FaultDetail(f.optString("ticketNo"))) } }
        }
        Spacer(Modifier.height(12.dp))
    }
}

private class FaultFormState {
    var faultType by mutableStateOf("no_internet")
    var address by mutableStateOf("")
    var desc by mutableStateOf("")
    var contact by mutableStateOf("")
}

@Composable
private fun FaultFormCard(
    scope: kotlinx.coroutines.CoroutineScope,
    nav: Nav, form: FaultFormState, onResult: (String) -> Unit,
) {
    AppCard {
        CardTitle("我要报修")
        FieldLabel("故障类型")
        TypeChips(form.faultType) { form.faultType = it }
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(value = form.address, onValueChange = { form.address = it }, label = { Text("故障地址") },
            singleLine = true, modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp))
        OutlinedTextField(value = form.desc, onValueChange = { form.desc = it }, label = { Text("故障描述") },
            placeholder = { Text("请描述故障现象") }, minLines = 3,
            modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp))
        OutlinedTextField(value = form.contact, onValueChange = { form.contact = it }, label = { Text("联系方式") },
            singleLine = true, modifier = Modifier.fillMaxWidth().padding(bottom = 10.dp))
        Button(
            onClick = { submitFault(scope, form, nav, onResult) },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().height(44.dp),
        ) { Text("提交报修") }
    }
}

private fun submitFault(
    scope: kotlinx.coroutines.CoroutineScope,
    form: FaultFormState, nav: Nav, onResult: (String) -> Unit,
) {
    if (form.address.isBlank() || form.desc.isBlank()) { onResult("请填写故障地址与描述"); return }
    scope.launch {
        try {
            val f = FaultApi.submit(form.faultType, form.address, form.desc, form.contact)
            onResult("已受理")
            val ticketNo = f.optString("ticketNo")
            if (ticketNo.isNotBlank()) nav.push(Route.FaultDetail(ticketNo))
        } catch (e: Exception) { onResult("提交失败，请稍后重试") }
    }
}

@Composable
private fun TypeChips(selected: String, onSelect: (String) -> Unit) {
    val types = listOf(
        "no_internet" to "无法上网", "slow" to "网速慢", "ont_fault" to "光猫故障", "other" to "其他",
    )
    Column {
        types.chunked(2).forEach { row ->
            Row(Modifier.fillMaxWidth().padding(bottom = 6.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                row.forEach { (k, label) -> Chip(Modifier.weight(1f), label, k == selected) { onSelect(k) } }
            }
        }
    }
}

@Composable
private fun Chip(modifier: Modifier, label: String, on: Boolean, onClick: () -> Unit) {
    Text(
        label, fontSize = 13.sp, textAlign = TextAlign.Center,
        color = if (on) Color.White else Palette.ink,
        modifier = modifier
            .background(if (on) Palette.primary else Palette.panel, RoundedCornerShape(8.dp))
            .border(1.dp, if (on) Palette.primary else Palette.line, RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp),
    )
}

@Composable
private fun QuickEntries(nav: Nav) {
    val entries = listOf(
        Triple(Icons.Outlined.Build, "自助排障") { nav.push(Route.Diy) },
        Triple(Icons.Outlined.HeadsetMic, "在线客服") { nav.push(Route.Service) },
    )
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        entries.forEachIndexed { i, (icon, label, onClick) ->
            val color = if (i == 0) Palette.primary else Palette.success
            Row(
                Modifier.weight(1f).background(Palette.panel, RoundedCornerShape(12.dp))
                    .clickable(onClick = onClick).padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconTile(icon, color, size = 36.dp, corner = 10.dp)
                Text(label, fontSize = 13.sp, color = Palette.ink, modifier = Modifier.padding(start = 8.dp))
            }
        }
    }
}

@Composable
private fun FaultCell(f: JSONObject, onClick: () -> Unit) {
    val resolved = f.optString("status") == "RESOLVED"
    CellRow(
        title = f.optString("faultTypeLabel"),
        desc = "${f.optString("ticketNo")} · ${f.optString("createdAt")}",
        onClick = onClick,
        right = { Tag(f.optString("statusLabel"), if (resolved) Palette.success else Palette.primary) },
    )
}
