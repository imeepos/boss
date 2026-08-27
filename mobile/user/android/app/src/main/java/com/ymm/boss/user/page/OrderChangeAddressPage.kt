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
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import com.ymm.boss.user.ui.rememberSubmitGuard
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应 OrderPage 底部"变更地址"入口:列出地址簿选一个,POST /orders/{no}/change-address。
@Composable
fun OrderChangeAddressScreen(nav: Nav, orderNo: String) {
    var items by remember { mutableStateOf<List<JSONObject>?>(null) }
    var selected by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    var done by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    // 防重复提交:弱网连点是重复提交地址变更。
    val guard = rememberSubmitGuard()

    LaunchedEffect(nav.refreshTick) {
        items = try {
            ProfileApi.addresses().optJSONArray("items").toObjectList()
        } catch (e: Exception) {
            err = "地址列表加载失败,请稍后重试"; emptyList()
        }
        val list = items
        if (selected.isBlank()) selected = list?.firstOrNull { it.optBoolean("isDefault") }
            ?.optString("addressId") ?: ""
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("变更安装地址", onBack = { nav.pop() })
        if (err.isNotBlank()) Notice(err, Palette.err)
        when {
            done -> DoneCard("安装地址已变更,师傅将按新地址上门", onBack = { nav.pop() })
            else -> {
                AddressPickCard(items, selected) { selected = it }
                Notice("仅装维中及之前状态的订单可变更地址;变更后师傅按新地址上门。")
                SubmitBar(
                    "确认变更地址", enabled = selected.isNotBlank() && !guard.active,
                    onSubmit = {
                        if (!guard.acquire()) return@SubmitBar
                        scope.launch {
                            try {
                                OrderApi.changeAddress(orderNo, selected)
                                done = true
                            } catch (e: Exception) { err = "变更失败,请稍后重试" } finally { guard.release() }
                        }
                    },
                )
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun AddressPickCard(items: List<JSONObject>?, selected: String, onSelect: (String) -> Unit) {
    AppCard {
        CardTitle("选择新地址")
        when {
            items == null -> Text("加载中…", fontSize = 13.sp, color = Palette.muted)
            items.isEmpty() -> EmptyState("暂无地址,请先到 地址管理 新增")
            else -> items.forEach { a ->
                val id = a.optString("addressId")
                CellRow(
                    title = a.optString("label").ifBlank { "未命名地址" },
                    desc = "${a.optString("contact")} · ${a.optString("phoneMasked")}",
                    onClick = { onSelect(id) },
                    right = {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Tag(
                                if (a.optBoolean("isDefault")) "默认" else "备用",
                                if (a.optBoolean("isDefault")) Palette.success else Palette.muted,
                            )
                            Spacer(Modifier.width(6.dp))
                            RadioButton(selected = selected == id, onClick = { onSelect(id) })
                        }
                    },
                )
            }
        }
    }
}
