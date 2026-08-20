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
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ComplaintApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/complaint.html:投诉建议表单 + 我的投诉列表。
@Composable
fun ComplaintScreen(nav: Nav) {
    val scope = rememberCoroutineScope()
    val form = remember { ComplaintFormState() }
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var msg by remember { mutableStateOf("") }

    fun reload() {
        scope.launch {
            try {
                items = ComplaintApi.list().optJSONArray("items").toObjList()
            } catch (e: Exception) {
                // 列表不可达时保留空骨架,与草稿一致
            }
        }
    }
    LaunchedEffect(nav.refreshTick) { reload() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("投诉与建议", onBack = { nav.pop() })
        ComplaintFormCard(scope, form) { msg = it; reload() }
        Notice(msg, if (msg.startsWith("已提交")) Palette.success else Palette.err)
        AppCard {
            CardTitle("我的投诉")
            if (items.isEmpty()) EmptyState("暂无投诉记录")
            items.forEach { c ->
                CellRow(
                    title = c.optString("typeLabel"),
                    desc = c.optString("createdAt"),
                    right = { Tag(c.optString("statusLabel", "已受理"), Palette.success) },
                )
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

private class ComplaintFormState {
    var type by mutableStateOf("attitude")
    var relOrderNo by mutableStateOf("")
    var desc by mutableStateOf("")
    var contact by mutableStateOf("")
}

@Composable
private fun ComplaintFormCard(
    scope: kotlinx.coroutines.CoroutineScope,
    form: ComplaintFormState, onResult: (String) -> Unit,
) {
    AppCard {
        CardTitle("我要投诉 / 建议")
        FieldLabel("类型")
        TypeChips(form.type) { form.type = it }
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(value = form.relOrderNo, onValueChange = { form.relOrderNo = it },
            label = { Text("关联订单 / 报修单（选填）") }, placeholder = { Text("如 ORD-20250817-001") },
            singleLine = true, modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp))
        OutlinedTextField(value = form.desc, onValueChange = { form.desc = it }, label = { Text("详细描述") },
            placeholder = { Text("请描述问题经过与诉求") }, minLines = 4,
            modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp))
        OutlinedTextField(value = form.contact, onValueChange = { form.contact = it }, label = { Text("联系方式") },
            singleLine = true, modifier = Modifier.fillMaxWidth().padding(bottom = 10.dp))
        Button(
            onClick = { submitComplaint(scope, form, onResult) },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().height(44.dp),
        ) { Text("提交投诉") }
        Notice("提交后 24 小时内受理，处理进度可在「消息中心」查询。")
    }
}

private fun submitComplaint(
    scope: kotlinx.coroutines.CoroutineScope,
    form: ComplaintFormState, onResult: (String) -> Unit,
) {
    if (form.desc.isBlank()) { onResult("请填写详细描述"); return }
    scope.launch {
        try {
            ComplaintApi.submit(form.type, form.relOrderNo, form.desc, form.contact)
            onResult("已提交，24 小时内受理")
        } catch (e: Exception) { onResult("提交失败，请稍后重试") }
    }
}

@Composable
private fun TypeChips(selected: String, onSelect: (String) -> Unit) {
    val types = listOf(
        "attitude" to "服务态度投诉", "quality" to "装维质量投诉", "billing" to "计费争议",
        "suggestion" to "业务建议", "other" to "其他",
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
