package com.ymm.boss.user.page

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
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
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Palette
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import org.json.JSONObject

/** 搜索触发下限:低于 2 字符不打远端,控请求成本(对辩折中案前提一)。 */
private const val MIN_QUERY_LEN = 2
private const val SEARCH_DEBOUNCE_MS = 300L
/** 后端 /address-tree/search LIMIT 20;命中上限时提示细化关键字。 */
private const val SEARCH_RESULT_CAP = 20

/** 首层搜索入口(显眼但非输入框):点开独立浮层输入,树形浏览仍是默认主视图。 */
@Composable
internal fun SearchEntryRow(onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth()
            .background(Palette.primary.copy(alpha = 0.08f), RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Outlined.Search, contentDescription = null,
            tint = Palette.primary, modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(
            "搜索 Barangay", fontSize = 14.sp, fontWeight = FontWeight.W600,
            color = Palette.primary, modifier = Modifier.weight(1f),
        )
        Text("知道街道名可直接搜", fontSize = 11.sp, color = Palette.subtle)
    }
}

/**
 * 地址搜索浮层:≥2 字符+300ms 防抖触发 /address-tree/search,query 再变即取消在途轮次。
 * 结果项=node 名+祖先链面包屑(菲律宾同名 Barangay 靠祖先消歧)。
 * node.hasChildren=false 叶命中直接选中回传;=true 以「祖先+该节点」为种子链下钻懒加载
 * (ancestors.hasChildren 恒 false 是后端已知缺陷,直选判定只认 node 自身)。
 * 自带 BackHandler:系统返回先回列表而非关闭选区弹层。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun AddressRegionSearchOverlay(
    onDismiss: () -> Unit,
    onPickLeaf: (RegionSelection) -> Unit,
    onDrillDown: (List<JSONObject>) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var query by remember { mutableStateOf("") }
    var results by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var searching by remember { mutableStateOf(false) }
    var searchErr by remember { mutableStateOf("") }

    LaunchedEffect(query) {
        val q = query.trim()
        if (q.length < MIN_QUERY_LEN) {
            results = emptyList()
            searchErr = ""
            searching = false
            return@LaunchedEffect
        }
        searching = true
        delay(SEARCH_DEBOUNCE_MS)
        try {
            results = ProfileApi.addressTreeSearch(q)
            searchErr = ""
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            searchErr = "搜索失败，" + Api.friendlyMessage(e)
        } finally {
            searching = false
        }
    }

    // 后于选区层注册,系统返回优先关浮层回列表;浮层已关时选区层 BackHandler 接管。
    BackHandler { onDismiss() }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState, containerColor = Palette.panel) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(
                    value = query, onValueChange = { query = it },
                    placeholder = { Text("输入 Barangay / 市镇名（至少 2 字）", fontSize = 13.sp, color = Palette.subtle) },
                    singleLine = true, shape = RoundedCornerShape(10.dp),
                    modifier = Modifier.weight(1f),
                )
                Spacer(Modifier.width(8.dp))
                TextButton(onClick = onDismiss) { Text("取消") }
            }
            Spacer(Modifier.height(4.dp))
            Box(Modifier.fillMaxWidth().heightIn(max = 380.dp)) {
                when {
                    query.trim().length < MIN_QUERY_LEN ->
                        EmptyState("输入至少 2 个字开始搜索", modifier = Modifier.align(Alignment.Center))
                    searching -> Box(Modifier.align(Alignment.Center)) {
                        CircularProgressIndicator(color = Palette.primary, modifier = Modifier.size(28.dp))
                    }
                    searchErr.isNotBlank() -> EmptyState(searchErr, modifier = Modifier.align(Alignment.Center))
                    results.isEmpty() -> EmptyState("无匹配结果，换个关键字试试", modifier = Modifier.align(Alignment.Center))
                    else -> LazyColumn {
                        items(results, key = { it.optJSONObject("node")?.optLong("id") ?: -1L }) { hit ->
                            SearchResultRow(hit = hit, onClick = { onHitPicked(hit, onPickLeaf, onDrillDown) })
                        }
                        if (results.size >= SEARCH_RESULT_CAP) {
                            item {
                                Text(
                                    "仅显示前 $SEARCH_RESULT_CAP 条，请细化关键字",
                                    fontSize = 12.sp, color = Palette.subtle,
                                    modifier = Modifier.fillMaxWidth().padding(vertical = 10.dp),
                                )
                            }
                        }
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
        }
    }
}

/** 叶命中→组装选中回传;非叶命中→祖先+节点为种子链,交还级联层继续懒加载下钻。 */
private fun onHitPicked(
    hit: JSONObject,
    onPickLeaf: (RegionSelection) -> Unit,
    onDrillDown: (List<JSONObject>) -> Unit,
) {
    val node = hit.optJSONObject("node") ?: JSONObject()
    if (node.optBoolean("hasChildren")) {
        onDrillDown(hit.optJSONArray("ancestors").toObjectList() + node)
    } else {
        onPickLeaf(selectionOfSearchHit(hit))
    }
}

/** 搜索命中→选点结果:addressPath=node.path,面包屑=祖先名+节点名(同名消歧关键)。 */
internal fun selectionOfSearchHit(hit: JSONObject): RegionSelection {
    val node = hit.optJSONObject("node") ?: JSONObject()
    val names = hit.optJSONArray("ancestors").toObjectList().map { it.optString("name") } +
            node.optString("name")
    return RegionSelection(
        addressPath = node.optString("path"),
        community = node.optString("name"),
        breadcrumb = names.filter { it.isNotBlank() }.joinToString(" · "),
    )
}

@Composable
private fun SearchResultRow(hit: JSONObject, onClick: () -> Unit) {
    val node = hit.optJSONObject("node") ?: JSONObject()
    val crumb = hit.optJSONArray("ancestors").toObjectList()
        .map { it.optString("name") }.filter { it.isNotBlank() }.joinToString(" · ")
    Column(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 10.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
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
        if (crumb.isNotBlank()) {
            Text(
                crumb, fontSize = 11.5.sp, color = Palette.subtle,
                modifier = Modifier.padding(start = 24.dp, top = 2.dp),
            )
        }
    }
}
