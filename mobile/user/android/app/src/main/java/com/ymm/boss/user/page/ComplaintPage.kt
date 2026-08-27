package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.ComplaintApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

private const val PAGE_SIZE = 10

/**
 * 投诉与建议主页:右上"立即投诉"按钮 → 弹层表单 → 提交成功重拉;
 * 主区列表接入分页(下拉刷新由全局 PageRefresh 驱动,上拉分页 PAGE_SIZE=10)。
 * 卡片点击进详情页(独立 ComplaintDetailScreen)。
 */
@Composable
fun ComplaintScreen(nav: Nav) {
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var page by remember { mutableIntStateOf(1) }
    var hasMore by remember { mutableStateOf(true) }
    var loading by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var submitted by remember { mutableStateOf("") }
    var showForm by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    suspend fun load(p: Int, replace: Boolean) {
        loading = true
        try {
            val resp = ComplaintApi.list(p, PAGE_SIZE)
            val chunk = resp.optJSONArray("items").toObjList()
            items = if (replace) chunk else items + chunk
            page = p
            hasMore = resp.optBoolean("hasMore", false)
            err = ""
        } catch (e: Exception) {
            err = "投诉记录加载失败," + Api.friendlyMessage(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(nav.refreshTick) { if (nav.refreshTick > 0) load(1, replace = true) }

    val listState = rememberLazyListState()
    val nearEnd by remember {
        derivedStateOf {
            val info = listState.layoutInfo
            val last = info.visibleItemsInfo.lastOrNull()?.index ?: -1
            info.totalItemsCount > 0 && last >= info.totalItemsCount - 2
        }
    }
    LaunchedEffect(nearEnd, page) {
        if (nearEnd && hasMore && !loading) load(page + 1, replace = false)
    }

    Column(Modifier.fillMaxSize()) {
        TopBar("投诉与建议", onBack = { nav.pop() }, action = "立即投诉", onAction = { showForm = true })
        if (submitted.isNotEmpty()) Notice(submitted, Palette.success)
        if (err.isNotEmpty()) Notice(err, Palette.err)
        LazyColumn(state = listState) {
            if (items.isEmpty() && err.isEmpty() && !loading) item { EmptyState("暂无投诉记录,点右上角发起") }
            items(items, key = { it.optString("complaintId") }) { c ->
                ComplaintCard(c) { nav.push(com.ymm.boss.user.ui.Route.ComplaintDetail(c.optString("complaintId"))) }
            }
            item {
                Text(
                    when {
                        loading && items.isNotEmpty() -> "加载中…"
                        hasMore -> ""
                        else -> "没有更多了"
                    },
                    fontSize = 12.sp, color = Palette.subtle,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
                    textAlign = TextAlign.Center,
                )
            }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }

    if (showForm) {
        ComplaintFormDialog(
            onDismiss = { showForm = false },
            onSubmit = { type, description, contact, relOrderNo ->
                showForm = false
                scope.launch {
                    try {
                        ComplaintApi.submit(type, description, contact, relOrderNo)
                        // 成功终态文案,禁止静默成功(木子红线:流程结束必须终态)
                        submitted = "投诉已提交,我们将尽快核实处理"
                        load(1, replace = true)
                    } catch (e: Exception) { err = "提交失败," + Api.friendlyMessage(e) }
                }
            },
        )
    }
}

@Composable
private fun ComplaintCard(c: JSONObject, onClick: () -> Unit) {
    val id = c.optString("complaintId")
    val typeLabel = c.optString("typeLabel", "其他")
    val status = c.optString("status", "OPEN")
    val statusLabel = c.optString("statusLabel", "受理中")
    val createdAt = c.optString("createdAt")
    val desc = c.optString("description")
    Column(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 14.dp, vertical = 6.dp)
            .background(Palette.panel, RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(14.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                id, fontSize = 13.sp, lineHeight = 18.sp, color = Palette.muted,
                modifier = Modifier.weight(1f),
            )
            Tag(statusLabel, statusTagColor(status))
        }
        Spacer(Modifier.height(6.dp))
        Text(typeLabel, fontSize = 15.sp, lineHeight = 20.sp, fontWeight = FontWeight.W600, color = Palette.ink)
        if (desc.isNotBlank()) {
            Spacer(Modifier.height(4.dp))
            Text(desc, fontSize = 13.sp, lineHeight = 18.sp, color = Palette.muted, maxLines = 2)
        }
        if (createdAt.isNotBlank()) {
            Spacer(Modifier.height(8.dp))
            Text(createdAt, fontSize = 11.5.sp, lineHeight = 14.sp, color = Palette.subtle)
        }
    }
}

private fun statusTagColor(status: String) = when (status) {
    "CLOSED" -> Palette.muted
    "PROCESSING" -> Palette.warn
    else -> Palette.primary
}

/**
 * 投诉表单 dialog:类型 5 选 1 + 关联订单/报修单 + 详细描述(>=4 字) + 联系方式。
 * 描述为空时不通过前端校验;真实校验由后端 binding 兜底。
 */
@Composable
private fun ComplaintFormDialog(
    onDismiss: () -> Unit,
    onSubmit: (type: String, description: String, contact: String?, relOrderNo: String?) -> Unit,
) {
    val types = remember {
        listOf(
            "attitude" to "服务态度投诉",
            "quality" to "装维质量投诉",
            "billing" to "计费争议",
            "suggestion" to "业务建议",
            "other" to "其他",
        )
    }
    var type by remember { mutableStateOf(types.first().first) }
    var relOrderNo by remember { mutableStateOf("") }
    var desc by remember { mutableStateOf("") }
    var contact by remember { mutableStateOf("") }
    val canSubmit = desc.trim().length >= 4

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("我要投诉 / 建议") },
        text = {
            Column {
                Text("类型", fontSize = 13.sp, color = Palette.muted)
                Spacer(Modifier.height(8.dp))
                TypeChips5(type, types) { type = it }
                Spacer(Modifier.height(12.dp))
                Text("关联订单 / 报修单(选填)", fontSize = 13.sp, color = Palette.muted)
                Spacer(Modifier.height(6.dp))
                UnderlineField(relOrderNo, "如 ORD-20250817-001") { relOrderNo = it }
                Spacer(Modifier.height(12.dp))
                Text("详细描述(≥4 字)", fontSize = 13.sp, color = Palette.muted)
                Spacer(Modifier.height(6.dp))
                UnderlineField(desc, "请描述问题经过与诉求", minLines = 4) { desc = it }
                Spacer(Modifier.height(12.dp))
                Text("联系方式(选填)", fontSize = 13.sp, color = Palette.muted)
                Spacer(Modifier.height(6.dp))
                UnderlineField(contact, "手机或座机") { contact = it }
            }
        },
        confirmButton = {
            TextButton(
                onClick = { onSubmit(type, desc.trim(), contact.ifBlank { null }, relOrderNo.ifBlank { null }) },
                enabled = canSubmit,
            ) { Text("提交", color = if (canSubmit) Palette.primary else Palette.subtle) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("取消") } },
    )
}

@Composable
private fun TypeChips5(selected: String, types: List<Pair<String, String>>, onSelect: (String) -> Unit) {
    Column {
        types.chunked(2).forEach { row ->
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                row.forEach { (k, label) -> Chip5(Modifier.weight(1f), label, k == selected) { onSelect(k) } }
            }
            Spacer(Modifier.height(6.dp))
        }
    }
}

@Composable
private fun Chip5(modifier: Modifier, label: String, on: Boolean, onClick: () -> Unit) {
    Text(
        label, fontSize = 12.5.sp, textAlign = TextAlign.Center,
        color = if (on) androidx.compose.ui.graphics.Color.White else Palette.ink,
        modifier = modifier
            .background(if (on) Palette.primary else Palette.panel, RoundedCornerShape(8.dp))
            .border(1.dp, if (on) Palette.primary else Palette.line, RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 7.dp),
    )
}

/** 极简下划线输入:沿用 Widgets.kt 风格的 visual,但避免 OutlinedTextField 在 dialog 中过宽。 */
@Composable
private fun UnderlineField(
    value: String,
    placeholder: String,
    minLines: Int = 1,
    onValueChange: (String) -> Unit,
) {
    Box(
        Modifier
            .fillMaxWidth()
            .background(Palette.panel, RoundedCornerShape(8.dp))
            .padding(horizontal = 10.dp, vertical = 8.dp),
    ) {
        if (value.isBlank()) {
            Text(placeholder, fontSize = 13.sp, color = Palette.subtle)
        }
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            textStyle = TextStyle(fontSize = 14.sp, color = Palette.ink),
            cursorBrush = SolidColor(Palette.primary),
            minLines = minLines,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}