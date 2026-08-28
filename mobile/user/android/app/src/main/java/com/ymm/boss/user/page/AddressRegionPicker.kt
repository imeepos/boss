package com.ymm.boss.user.page

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.ModalBottomSheetProperties
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
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
    var searchVisible by remember { mutableStateOf(false) }
    // 错误重试触发器:reloadTick 变化即整链重载(网络瞬断后不必关弹层重开)。
    var reloadTick by remember { mutableIntStateOf(0) }

    // 返回键语义全接管:M3 弹层的内置返回处理(禁用前)与上滑/点遮罩统一走
    // onDismissRequest,内容里的 BackHandler 抢不过它,层2/层3 返回被整层关闭。
    // 故 properties 禁用内置返回后在此分层:链非空回上一级,链空才真关闭。
    // 不清 filter 会残留上一层的过滤词,回到下层时列表被误过滤。
    BackHandler {
        if (chain.isNotEmpty()) {
            chain.removeAt(chain.size - 1)
            filter = ""
        } else {
            onDismiss()
        }
    }

    LaunchedEffect(chain.size, reloadTick) { loadChildren(chain, setLoading = { loading = it }, setErr = { err = it }) { children = it } }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Palette.panel,
        // 禁用弹层内置返回关闭:返回键归上面的 BackHandler 分层处理;
        // 上滑拖拽与点遮罩不受影响,仍走 onDismissRequest 真关闭,不会被误判成回上一级。
        properties = ModalBottomSheetProperties(shouldDismissOnBackPress = false),
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            PickerHeader(
                chain = chain,
                onBack = { popped ->
                    if (popped) {
                        chain.removeAt(chain.size - 1)
                        filter = ""
                    } else {
                        onDismiss()
                    }
                },
                onJumpTo = { index ->
                    if (index in 0 until chain.size - 1) {
                        chain.subList(index + 1, chain.size).clear()
                        filter = ""
                    }
                },
            )
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
            // 首层搜索入口(对辩折中案):菲律宾用户多只知 Barangay 名,入口常驻但
            // 仍是入口样式,树形浏览保持默认主视图,搜索在独立浮层内完成。
            if (chain.isEmpty()) {
                SearchEntryRow(onClick = { searchVisible = true })
                Spacer(Modifier.height(8.dp))
            }
            // 高度自适应:内容少时弹层随内容收缩,上限 420dp 防超高(原固定 420 常留大片空白)。
            Box(Modifier.fillMaxWidth().heightIn(max = 420.dp)) {
                when {
                    loading -> Box(Modifier.align(Alignment.Center)) {
                        CircularProgressIndicator(color = Palette.primary, modifier = Modifier.size(28.dp))
                    }
                    err.isNotBlank() -> Column(Modifier.align(Alignment.Center)) {
                        EmptyState(err)
                        Spacer(Modifier.height(8.dp))
                        TextButton(onClick = { reloadTick++ }) { Text("重试") }
                    }
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
                                    NodeRow(node = node, enabled = !loading, onClick = {
                                        // 直选判定只认 hasChildren:level<5 硬编码在树加深/
                                        // 层级变动时会把叶子当中间层,hasChildren 才是权威信号。
                                        if (node.optBoolean("hasChildren")) {
                                            chain.add(node); filter = ""
                                        } else {
                                            onSelected(selectionOf(chain, node))
                                        }
                                    })
                                }
                            }
                        }
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
        }
    }

    // 搜索浮层:叶命中直接回传选区;非叶命中以「祖先+节点」重建 chain 继续懒加载下钻
    // (保留祖先链上下文,面包屑与后续选中结果不断层)。
    if (searchVisible) {
        AddressRegionSearchOverlay(
            onDismiss = { searchVisible = false },
            onPickLeaf = { sel ->
                searchVisible = false
                onSelected(sel)
            },
            onDrillDown = { seed ->
                searchVisible = false
                chain.clear()
                chain.addAll(seed)
                filter = ""
            },
        )
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
