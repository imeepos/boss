package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import org.json.JSONObject

// 对应设计稿 user-products-orders-profile.png 左屏(服务 tab):搜索 + 分类胶囊 + 产品卡。
// 骨架屏/搜索头/分类/产品卡等组件拆在 ProductsComponents.kt(≤300 行红线)。

@Composable
fun ProductsScreen(nav: Nav) {
    var cat by remember { mutableStateOf("broadband") }
    var query by remember { mutableStateOf("") }
    var products by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var addons by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    // 首屏/分类切换 loading:骨架屏替代误显空态(木子红线:产品卡首屏必须骨架屏)。
    var loading by remember { mutableStateOf(true) }
    LaunchedEffect(cat, nav.refreshTick) {
        loading = true
        try {
            val d = ProductApi.list(cat)
            products = d.optJSONArray("items").toObjList()
            addons = d.optJSONArray("addons").toObjList()
            err = ""
        } catch (e: Exception) { err = "套餐加载失败,请稍后重试" } finally { loading = false }
    }

    Column(Modifier.fillMaxSize()) {
        ServiceSearchHeader(query) { query = it }
        CategorySeg(cat) { cat = it }
        val shown = products.filter {
            query.isBlank() || it.optString("name").contains(query, true) ||
                it.optString("bandwidth").contains(query, true) ||
                it.optString("description").contains(query, true)
        }
        LazyColumn {
            if (loading) {
                item { SkeletonProducts() }
            } else {
                item { if (err.isNotEmpty()) Notice(err, Palette.err) }
                if (shown.isEmpty() && err.isEmpty()) item { EmptyState("未找到匹配的产品") }
                items(shown) { p -> ProductCard(p, nav) }
                if (cat == "addon") item { AddonCard(addons, nav) }
            }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }
}