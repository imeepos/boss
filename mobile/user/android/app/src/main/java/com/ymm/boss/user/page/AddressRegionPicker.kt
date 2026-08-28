package com.ymm.boss.user.page

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Palette
import kotlinx.coroutines.CancellationException
import org.json.JSONObject

/** 级联选点结果:path 存 user_addresses.address_path,community 自动填街道/小区名。 */
internal data class RegionSelection(
    val addressPath: String,
    val community: String,
    val breadcrumb: String,
)

/**
 * 地址层级级联选择弹层:大区/市 → 市 → 街道(Barangay)逐层懒加载,
 * 最深层带客户端过滤(单市 Barangay 上百,翻列表不如搜)。
 * 数据源 GET /address-tree?parentId=;选中即回传,不再下钻的层级点行直接选。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun AddressRegionPickerSheet(
    onDismiss: () -> Unit,
    onSelected: (RegionSelection) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    // 已选链:level 递增的节点;链长即当前层。
    val chain = remember { mutableStateListOf<JSONObject>() }
    var children by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var loading by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var filter by remember { mutableStateOf("") }

    // 系统返回=回上一级而非整层关闭;链空时不拦截,交还弹层自身关闭。
    // 不清 filter 会残留上一层的过滤词,回到下层时列表被误过滤。
    BackHandler(enabled = chain.isNotEmpty()) {
        chain.removeAt(chain.size - 1)
        filter = ""
    }

    LaunchedEffect(chain.size) { loadChildren(chain, setLoading = { loading = it }, setErr = { err = it }) { children = it } }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState, containerColor = Palette.panel) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            PickerHeader(chain) { popped ->
                if (popped) chain.removeAt(chain.size - 1) else onDismiss()
            }
            Spacer(Modifier.height(4.dp))
            val level = chain.size + 1
            if (level >= 3) {
                OutlinedTextField(
                    value = filter, onValueChange = { filter = it },
                    placeholder = { Text("输入关键字过滤", fontSize = 13.sp, color = Palette.subtle) },
                    singleLine = true, shape = RoundedCornerShape(10.dp),
                    modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                )
            }
            Box(Modifier.height(420.dp)) {
                when {
                    loading -> Box(Modifier.align(Alignment.Center)) {
                        CircularProgressIndicator(color = Palette.primary, modifier = Modifier.size(28.dp))
                    }
                    err.isNotBlank() -> EmptyState(err, modifier = Modifier.align(Alignment.Center))
                    else -> {
                        val shown = if (level >= 3 && filter.isNotBlank()) {
                            children.filter { it.optString("name").contains(filter, ignoreCase = true) }
                        } else children
                        if (shown.isEmpty()) {
                            EmptyState(if (filter.isBlank()) "暂无可选节点" else "无匹配结果",
                                modifier = Modifier.align(Alignment.Center))
                        } else {
                            LazyColumn {
                                items(shown, key = { it.optLong("id") }) { node ->
                                    NodeRow(node) {
                                        val hasNext = node.optBoolean("hasChildren")
                                        if (hasNext && node.optInt("level") < 5) {
                                            chain.add(node); filter = ""
                                        } else {
                                            onSelected(selectionOf(chain, node))
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
        }
    }
}

private suspend fun loadChildren(
    chain: List<JSONObject>,
    setLoading: (Boolean) -> Unit,
    setErr: (String) -> Unit,
    setChildren: (List<JSONObject>) -> Unit,
) {
    setLoading(true)
    try {
        val parentId = chain.lastOrNull()?.optLong("id") ?: 0L
        setChildren(ProfileApi.addressTree(parentId))
        setErr("")
    } catch (e: CancellationException) {
        throw e
    } catch (e: Exception) {
        setErr("地区加载失败，" + Api.friendlyMessage(e))
    } finally {
        setLoading(false)
    }
}

/** 组装选点结果:面包屑=祖先名串联,community=最深层节点名。 */
private fun selectionOf(chain: List<JSONObject>, leaf: JSONObject): RegionSelection =
    RegionSelection(
        addressPath = leaf.optString("path"),
        community = leaf.optString("name"),
        breadcrumb = (chain.map { it.optString("name") } + leaf.optString("name")).joinToString(" · "),
    )

@Composable
private fun PickerHeader(chain: List<JSONObject>, onBack: (Boolean) -> Unit) {
    val title = when (chain.size) {
        0 -> "选择大区 / 市"
        1 -> "选择城市 / 市镇"
        2 -> "选择街道 (Barangay)"
        else -> "选择小区 / 楼栋"
    }
    Row(
        Modifier.fillMaxWidth().padding(bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (chain.isNotEmpty()) {
            Icon(
                Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回上一级",
                tint = Palette.muted, modifier = Modifier.size(20.dp).clickable { onBack(true) },
            )
        }
        Text(title, fontSize = 16.sp, fontWeight = FontWeight.W600, color = Palette.ink)
    }
}

@Composable
private fun NodeRow(node: JSONObject, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Outlined.LocationOn, contentDescription = null,
            tint = Palette.primary.copy(alpha = 0.7f), modifier = Modifier.size(16.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(node.optString("name"), fontSize = 14.sp, color = Palette.ink, modifier = Modifier.weight(1f))
        if (node.optBoolean("hasChildren")) {
            Text("›", fontSize = 16.sp, color = Palette.subtle)
        }
    }
}
