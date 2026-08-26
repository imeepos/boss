package com.ymm.boss.user.page

import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.toObjectList
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
 * 家庭地址管理:顶部 48dp 蓝带 + 右上"新增"→ 编辑弹窗;
 * 列表卡(默认地址主标 + 次行联系/详细 + 编辑/删除操作);
 * 下拉刷新由全局 PageRefresh 驱动(nav.refreshTick);上拉分页 pageSize=10。
 */
@Composable
fun AddressScreen(nav: Nav) {
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var page by remember { mutableIntStateOf(1) }
    var hasMore by remember { mutableStateOf(true) }
    var loading by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var editor by remember { mutableStateOf<EditorTarget?>(null) }
    var pendingDelete by remember { mutableStateOf<JSONObject?>(null) }
    val scope = rememberCoroutineScope()

    suspend fun load(p: Int, replace: Boolean) {
        loading = true
        try {
            val resp = ProfileApi.addresses(p, PAGE_SIZE)
            val chunk = resp.optJSONArray("items").toObjectList()
            items = if (replace) chunk else items + chunk
            page = p
            hasMore = resp.optBoolean("hasMore", false)
            err = ""
        } catch (e: Exception) {
            err = "地址加载失败," + Api.friendlyMessage(e)
        } finally {
            loading = false
        }
    }

    // nav.refreshTick=0 是初始合成态,跳过避免空触发;>0 表示全局下拉刷新一次。
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
        TopBar("家庭地址管理", onBack = { nav.pop() }, action = "新增", onAction = {
            editor = EditorTarget(null)
        })
        if (err.isNotEmpty()) Notice(err, Palette.err)
        LazyColumn(state = listState) {
            if (items.isEmpty() && err.isEmpty() && !loading) item { EmptyState("暂无地址,点右上角新增") }
            items(items, key = { it.optString("addressId") }) { a ->
                AddressCard(a,
                    onEdit = { editor = EditorTarget(a) },
                    onDelete = { pendingDelete = a },
                )
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

    editor?.let { target ->
        AddressEditorSheet(
            initial = target.existing,
            onDismiss = { editor = null },
            onSubmit = { payload ->
                editor = null
                scope.launch {
                    try {
                        if (target.existing == null) ProfileApi.createAddress(payload)
                        else ProfileApi.updateAddress(target.existing.optString("addressId"), payload)
                        load(1, replace = true)
                    } catch (e: Exception) { err = "保存失败," + Api.friendlyMessage(e) }
                }
            },
        )
    }
    pendingDelete?.let { a ->
        val id = a.optString("addressId")
        AlertDialog(
            onDismissRequest = { pendingDelete = null },
            title = { Text("删除地址") },
            text = { Text("确定删除 「${a.optString("label").ifBlank { "该地址" }}」?删除后无法恢复。") },
            confirmButton = {
                TextButton(onClick = {
                    pendingDelete = null
                    scope.launch {
                        try {
                            ProfileApi.deleteAddress(id)
                            items = items.filterNot { it.optString("addressId") == id }
                        } catch (e: Exception) { err = "删除失败," + Api.friendlyMessage(e) }
                    }
                }) { Text("删除", color = Palette.err) }
            },
            dismissButton = { TextButton(onClick = { pendingDelete = null }) { Text("取消") } },
        )
    }
}

private data class EditorTarget(val existing: JSONObject?)

@Composable
private fun AddressCard(a: JSONObject, onEdit: () -> Unit, onDelete: () -> Unit) {
    val isDefault = a.optBoolean("isDefault")
    val label = a.optString("label").ifBlank { "未命名地址" }
    Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 14.dp, vertical = 6.dp)
            .background(Palette.panel, RoundedCornerShape(12.dp))
            .padding(14.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            Modifier
                .size(36.dp)
                .background(Palette.primary.copy(alpha = 0.1f), RoundedCornerShape(10.dp)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                if (isDefault) Icons.Outlined.Home else Icons.Outlined.LocationOn,
                contentDescription = null, tint = Palette.primary, modifier = Modifier.size(20.dp),
            )
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    label, fontSize = 15.sp, lineHeight = 20.sp, fontWeight = FontWeight.W600,
                    color = Palette.ink, modifier = Modifier.weight(1f),
                )
                Tag(if (isDefault) "默认" else "备用", if (isDefault) Palette.success else Palette.muted)
            }
            Spacer(Modifier.height(4.dp))
            Text(
                "${a.optString("contact")} · ${a.optString("phoneMasked")}",
                fontSize = 12.5.sp, lineHeight = 16.sp, color = Palette.muted,
            )
            val sub = listOf(a.optString("community"), a.optString("building"), a.optString("door"))
                .filter { it.isNotBlank() }.joinToString(" ")
            if (sub.isNotBlank()) {
                Text(sub, fontSize = 12.5.sp, lineHeight = 16.sp, color = Palette.muted,
                    modifier = Modifier.padding(top = 2.dp))
            }
            Spacer(Modifier.height(10.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                ActionChip(Icons.Outlined.Edit, "编辑", onEdit)
                ActionChip(Icons.Outlined.Delete, "删除", onDelete, danger = true)
            }
        }
    }
}

@Composable
private fun ActionChip(icon: ImageVector, label: String, onClick: () -> Unit, danger: Boolean = false) {
    val tint = if (danger) Palette.err else Palette.primary
    Row(
        Modifier
            .background(tint.copy(alpha = 0.1f), RoundedCornerShape(999.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = label, tint = tint, modifier = Modifier.size(14.dp))
        Spacer(Modifier.width(4.dp))
        Text(label, fontSize = 12.sp, color = tint)
    }
}
