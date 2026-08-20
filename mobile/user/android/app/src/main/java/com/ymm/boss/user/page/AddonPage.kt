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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CardGiftcard
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
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
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 对应草稿 docs/user/addon.html:增值包可订购列表 + 已订购列表,订阅/退订后刷新。
// 契约: GET /addons {available[],subscribed[]}; POST /addons/{addonId}/{subscribe,unsubscribe}
// 视觉基准与 ProductsPage 增值服务卡对齐:IconTile + 名称 14sp W500 + 金额 Bold。
@Composable
fun AddonScreen(nav: Nav) {
    var available by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var subscribed by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var msg by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val reload: () -> Unit = {
        scope.launch {
            try { loadAddons({ available = it.first; subscribed = it.second }) }
            catch (e: Exception) { err = "增值服务加载失败,请稍后重试" }
        }
    }
    LaunchedEffect(nav.refreshTick) { reload() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("加购增值服务", onBack = { nav.pop() })
        if (err.isNotBlank()) Notice(err, Palette.err)
        if (msg.isNotBlank()) Notice(msg, Palette.success)
        AddonListCard("可订购增值服务", available, subscribed = false, onToggle = { id, name ->
            scope.launch { toggleAddon(id, true, name, { msg = it; err = "" }, { err = it }, reload) }
        })
        AddonListCard("已订购", subscribed, subscribed = true, onToggle = { id, name ->
            scope.launch { toggleAddon(id, false, name, { msg = it; err = "" }, { err = it }, reload) }
        })
        Spacer(Modifier.height(12.dp))
    }
}

private suspend fun loadAddons(set: (Pair<List<JSONObject>, List<JSONObject>>) -> Unit) {
    val d = PlanApi.addons()
    val a = d.optJSONArray("available").toList()
    val s = d.optJSONArray("subscribed").toList()
    set(a to s)
}

private suspend fun toggleAddon(
    addonId: String, subscribe: Boolean, name: String,
    onDone: (String) -> Unit, onErr: (String) -> Unit, reload: () -> Unit,
) {
    try {
        if (subscribe) PlanApi.subscribe(addonId) else PlanApi.unsubscribe(addonId)
        onDone((if (subscribe) "已订购 " else "已退订 ") + name)
        reload()
    } catch (e: Exception) { onErr(if (subscribe) "订购失败,请稍后重试" else "退订失败,请稍后重试") }
}

@Composable
private fun AddonListCard(
    title: String, list: List<JSONObject>, subscribed: Boolean,
    onToggle: (String, String) -> Unit,
) {
    AppCard {
        Column(Modifier.fillMaxWidth()) {
            CardTitle(title)
            if (list.isEmpty()) EmptyState(if (subscribed) "暂未订购增值服务" else "暂无可订购增值服务")
            list.forEach { a -> AddonCell(a, subscribed, onAction = { onToggle(a.optString("addonId"), a.optString("name")) }) }
        }
    }
}

/** 行形态与 ProductsPage.AddonCard 一致:IconTile 紫色 + 名称/描述 + 右侧金额 Bold 与操作按钮。 */
@Composable
private fun AddonCell(a: JSONObject, subscribed: Boolean, onAction: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
        IconTile(Icons.Filled.CardGiftcard, Palette.purple, size = 36.dp)
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(a.optString("name"), fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink)
            if (a.optString("description").isNotEmpty()) {
                Text(a.optString("description"), fontSize = 12.sp, color = Palette.muted)
            }
        }
        Column(horizontalAlignment = Alignment.End) {
            Text("¥${a.optString("monthlyFee")}/月", fontSize = 13.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
            Spacer(Modifier.height(6.dp))
            if (subscribed) {
                OutlinedButton(onClick = onAction, modifier = Modifier.height(32.dp)) { Text("退订", fontSize = 13.sp) }
            } else {
                Button(
                    onClick = onAction,
                    colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                    modifier = Modifier.height(32.dp),
                ) { Text("订购", fontSize = 13.sp) }
            }
        }
    }
}
