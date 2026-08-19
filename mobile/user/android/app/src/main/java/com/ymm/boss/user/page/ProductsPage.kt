package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

private const val DEMO_ADDRESS_ID = "ADDR-001"

// 对应草稿 docs/user/products.html:套餐列表 + 分类筛选 + 增值服务。tab 页。
@Composable
fun ProductsScreen(nav: Nav) {
    var cat by remember { mutableStateOf("broadband") }
    var products by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var addons by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    LaunchedEffect(cat) {
        try {
            val d = ProductApi.list(cat)
            products = d.optJSONArray("items").toObjList()
            addons = d.optJSONArray("addons").toObjList()
            err = ""
        } catch (e: Exception) { err = "套餐加载失败,请稍后重试" }
    }

    Column(Modifier.fillMaxSize()) {
        TopBar("产品套餐")
        CategorySeg(cat) { cat = it }
        LazyColumn {
            item { if (err.isNotEmpty()) Notice(err, Palette.err) }
            items(products) { p -> ProductCard(p, nav) }
            item { AddonCard(addons, nav) }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }
}

@Composable
private fun CategorySeg(current: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(18.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf("broadband" to "宽带套餐", "fusion" to "融合套餐", "addon" to "增值服务").forEach { (k, label) ->
            val active = k == current
            Text(
                label, fontSize = 13.5.sp,
                color = if (active) Palette.primary else Palette.muted,
                fontWeight = if (active) FontWeight.Bold else FontWeight.Normal,
                modifier = Modifier.clickable { onSelect(k) },
            )
        }
    }
}

@Composable
private fun ProductCard(p: JSONObject, nav: Nav) {
    val scope = rememberCoroutineScope()
    val id = p.optString("productId")
    AppCard {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(p.optString("name"), fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink)
            Tag("¥${p.optString("monthlyFee")}/月", if (p.optBoolean("featured")) Palette.orange else Palette.primary)
        }
        Notice(p.optString("description"))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(onClick = { nav.push(Route.Product(id)) }) { Text("详情", color = Palette.primary) }
            Button(
                onClick = { buyNow(scope, nav, id) },
                colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            ) { Text("立即办理") }
        }
    }
}

@Composable
private fun AddonCard(addons: List<JSONObject>, nav: Nav) {
    AppCard {
        CardTitle("增值服务", more = "进入管理 >") { nav.push(Route.Addon) }
        if (addons.isEmpty()) Notice("暂无可订购增值服务")
        addons.forEach { a ->
            Row(Modifier.padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(a.optString("name"), fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink)
                    Notice(a.optString("description"))
                }
                Text("¥${a.optString("monthlyFee")}/月", fontSize = 12.sp, color = Palette.muted)
            }
        }
    }
}

private fun buyNow(scope: kotlinx.coroutines.CoroutineScope, nav: Nav, productId: String) {
    scope.launch {
        try {
            val o = OrderApi.submit(productId, DEMO_ADDRESS_ID)
            nav.push(Route.Order(o.optString("orderNo")))
        } catch (e: Exception) { } // 下单失败静默,与草稿 catch 行为一致
    }
}
