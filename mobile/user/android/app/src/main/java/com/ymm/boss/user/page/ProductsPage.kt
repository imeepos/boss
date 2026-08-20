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
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.CardGiftcard
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.SignalCellularAlt
import androidx.compose.material.icons.filled.Speed
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.statusBarSolid
import org.json.JSONObject

// 对应设计稿 user-products-orders-profile.png 左屏(服务 tab):搜索 + 分类胶囊 + 产品卡。
@Composable
fun ProductsScreen(nav: Nav) {
    var cat by remember { mutableStateOf("broadband") }
    var query by remember { mutableStateOf("") }
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
        ServiceSearchHeader(query) { query = it }
        CategorySeg(cat) { cat = it }
        val shown = products.filter {
            query.isBlank() || it.optString("name").contains(query, true) ||
                it.optString("bandwidth").contains(query, true) ||
                it.optString("description").contains(query, true)
        }
        LazyColumn {
            item { if (err.isNotEmpty()) Notice(err, Palette.err) }
            if (shown.isEmpty() && err.isEmpty()) item { Notice("未找到匹配的产品") }
            items(shown) { p -> ProductCard(p, nav) }
            if (cat == "addon") item { AddonCard(addons, nav) }
            item { Spacer(Modifier.height(12.dp)) }
        }
    }
}

/** 服务页 Header:无标题,搜索框直接嵌入 48dp 状态栏色顶栏(半透明白胶囊)。 */
@Composable
private fun ServiceSearchHeader(value: String, onChange: (String) -> Unit) {
    Row(
        Modifier.fillMaxWidth().background(statusBarSolid()).padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(
            Modifier.fillMaxWidth()
                .background(Color.White.copy(alpha = 0.18f), RoundedCornerShape(999.dp))
                .padding(horizontal = 14.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Filled.Search, contentDescription = null, tint = Color.White, modifier = Modifier.size(16.dp))
            Spacer(Modifier.width(8.dp))
            BasicTextField(
                value = value, onValueChange = onChange, singleLine = true,
                textStyle = TextStyle(fontSize = 13.sp, color = Color.White),
                cursorBrush = SolidColor(Color.White),
                modifier = Modifier.weight(1f),
                decorationBox = { inner ->
                    if (value.isEmpty()) Text("搜索产品或服务", fontSize = 13.sp, color = Color.White.copy(alpha = 0.7f))
                    inner()
                },
            )
        }
    }
}

@Composable
private fun CategorySeg(current: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.padding(horizontal = 14.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf(
            Triple("broadband", "宽带", Icons.Filled.Wifi),
            Triple("fusion", "5G", Icons.Filled.SignalCellularAlt),
            Triple("addon", "增值服务", Icons.Filled.CardGiftcard),
        ).forEach { (k, label, icon) ->
            PillTab(label, active = k == current, onClick = { onSelect(k) }, icon = icon)
        }
    }
}

@Composable
private fun ProductCard(p: JSONObject, nav: Nav) {
    val id = p.optString("productId")
    AppCard(Modifier.clickable { nav.push(Route.Product(id)) }) {
        Column(Modifier.fillMaxWidth()) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    p.optString("name"), fontSize = 15.sp, fontWeight = FontWeight.W600,
                    color = Palette.ink, modifier = Modifier.weight(1f),
                )
                if (p.optBoolean("featured")) Tag("热门", Palette.success)
                Spacer(Modifier.width(10.dp))
                PricePill(p.optString("monthlyFee"), recommended = p.optString("bandwidth").contains("500"))
            }
            Spacer(Modifier.height(10.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                IconTile(Icons.Filled.Wifi, Palette.primary, size = 44.dp, corner = 10.dp)
                Spacer(Modifier.width(10.dp))
                Column(Modifier.weight(1f)) {
                    MetaLine(p)
                    if (p.optString("description").isNotEmpty()) {
                        FeatureLine(Icons.Filled.CheckCircle, p.optString("description"))
                    }
                }
                Icon(
                    Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = "查看详情",
                    tint = Palette.muted,
                    modifier = Modifier.padding(start = 4.dp).size(20.dp),
                )
            }
        }
    }
}

/** 右上角价格胶囊:推荐档橙色底+右上角拇指角标,普通档品牌蓝浅底。 */
@Composable
private fun PricePill(fee: String, recommended: Boolean) {
    val bg = if (recommended) Palette.warn else Palette.primary.copy(alpha = 0.12f)
    val fg = if (recommended) Color.White else Palette.primary
    Box {
        Row(
            verticalAlignment = Alignment.Bottom,
            modifier = Modifier
                .background(bg, RoundedCornerShape(8.dp))
                .padding(horizontal = 10.dp, vertical = 4.dp),
        ) {
            Text("¥", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = fg)
            Text(fee, fontSize = 16.sp, fontWeight = FontWeight.Bold, color = fg)
            Text("/月", fontSize = 10.sp, color = fg, modifier = Modifier.padding(bottom = 1.dp))
        }
        if (recommended) {
            Icon(
                Icons.Filled.ThumbUp,
                contentDescription = "推荐",
                tint = Palette.warn,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .offset(x = 7.dp, y = (-7).dp)
                    .background(Color.White, CircleShape)
                    .padding(2.dp)
                    .size(10.dp),
            )
        }
    }
}

@Composable
private fun MetaLine(p: JSONObject) {
    val contract = p.optInt("contractMonths")
    val feats = buildList {
        add(p.optString("bandwidth").ifBlank { "高速带宽" })
        if (contract > 0) add("含合约${contract}个月")
    }.joinToString(" · ")
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(Icons.Filled.Speed, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(14.dp))
        Spacer(Modifier.width(6.dp))
        Text(feats, fontSize = 12.sp, color = Palette.muted, modifier = Modifier.weight(1f), maxLines = 1)
    }
}

@Composable
private fun FeatureLine(icon: androidx.compose.ui.graphics.vector.ImageVector, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 3.dp)) {
        Icon(icon, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(14.dp))
        Spacer(Modifier.width(6.dp))
        Text(text, fontSize = 12.sp, color = Palette.muted)
    }
}

@Composable
private fun AddonCard(addons: List<JSONObject>, nav: Nav) {
    AppCard {
        Column(Modifier.fillMaxWidth()) {
            CardTitle("增值服务", more = "进入管理 >") { nav.push(Route.Addon) }
            if (addons.isEmpty()) Notice("暂无可订购增值服务")
            addons.forEach { a ->
                Row(Modifier.padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    IconTile(Icons.Filled.CardGiftcard, Palette.purple, size = 36.dp)
                    Spacer(Modifier.width(12.dp))
                    Column(Modifier.weight(1f)) {
                        Text(a.optString("name"), fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink)
                        if (a.optString("description").isNotEmpty()) {
                            Text(a.optString("description"), fontSize = 12.sp, color = Palette.muted)
                        }
                    }
                    Text("¥${a.optString("monthlyFee")}/月", fontSize = 12.sp, color = Palette.muted)
                }
            }
        }
    }
}
