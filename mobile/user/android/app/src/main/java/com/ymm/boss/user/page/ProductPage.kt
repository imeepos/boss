package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.Surface
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PricePill
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 套餐详情页。视觉基准 designs/product-detail-v1.png:TopBar + 主信息卡 + 规格卡 +
// 对比卡(含"当前套餐"徽章)+ 合约说明卡 + 底部固定 CTA 通栏按钮。
// 价格胶囊复用 PricePill,卡片复用 AppCard/CardTitle/CellRow/EmptyState/Notice,Tag 复用 Widgets.kt。
// CTA:点击"立即办理" → 跳 OrderConfirmScreen 选地址 + Stripe 支付(原逻辑直接 POST /orders
// 用 DEMO_ADDRESS_ID="ADDR-001",backend portalID 解析失败导致"下单失败,请稍后重试")。
@Composable
fun ProductScreen(nav: Nav, id: String) {
    var product by remember { mutableStateOf<JSONObject?>(null) }
    var specs by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var compare by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    LaunchedEffect(id, nav.refreshTick) {
        try {
            val d = ProductApi.detail(id)
            product = d.optJSONObject("product")
            specs = d.optJSONArray("specs").toObjList()
            compare = d.optJSONArray("compare").toObjList()
            err = ""
        } catch (e: Exception) { err = "套餐详情加载失败" }
    }

    Box(Modifier.fillMaxSize().background(Palette.bg)) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
            TopBar("套餐详情", onBack = { nav.pop() })
            if (err.isNotEmpty()) Notice(err, Palette.err)
            DetailCard(product)
            if (specs.isNotEmpty()) SpecsCard(specs)
            CompareCard(compare, id, nav)
            ContractCard()
            Spacer(Modifier.height(96.dp))
        }
        CtaBar(product = product, onClick = { nav.push(Route.OrderConfirm(id)) })
    }
}

/** 产品主信息卡:名称 + 可选热门 Tag + 价格胶囊 + 描述。 */
@Composable
private fun DetailCard(product: JSONObject?) {
    val name = product?.optString("name")?.takeIf { it.isNotBlank() } ?: "加载中…"
    val fee = product?.optString("monthlyFee")?.takeIf { it.isNotBlank() } ?: "—"
    val desc = product?.optString("description")?.takeIf { it.isNotBlank() } ?: ""
    val featured = product?.optBoolean("featured") == true
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                name, fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink,
                modifier = Modifier.weight(1f),
            )
            if (featured) {
                Tag("热门", Palette.success)
                Spacer(Modifier.width(8.dp))
            }
            PricePill(fee)
        }
        if (desc.isNotEmpty()) Notice(desc)
    }
}

/** 套餐规格卡:逐行 label-value 对(复用 CellRow)。 */
@Composable
private fun SpecsCard(specs: List<JSONObject>) {
    AppCard {
        CardTitle("套餐规格")
        specs.forEach { s ->
            val label = s.optString("label")
            if (label.isNotEmpty()) {
                CellRow(title = label, right = {
                    Text(s.optString("value").ifBlank { "—" }, fontSize = 13.sp, color = Palette.muted)
                })
            }
        }
    }
}

/** 套餐对比卡:同类套餐列表,当前产品以"当前套餐"徽章标记。 */
@Composable
private fun CompareCard(compare: List<JSONObject>, currentId: String, nav: Nav) {
    AppCard {
        CardTitle("套餐对比", more = if (compare.isNotEmpty()) "同档可选" else null)
        if (compare.isEmpty()) {
            EmptyState("暂无可比套餐")
        } else {
            compare.forEach { p -> CompareRow(p, currentId, nav) }
        }
    }
}

@Composable
private fun CompareRow(p: JSONObject, currentId: String, nav: Nav) {
    val pid = p.optString("productId")
    val isCurrent = pid.isNotEmpty() && pid == currentId
    val rowMod = if (isCurrent) Modifier.fillMaxWidth().padding(vertical = 12.dp)
    else Modifier.fillMaxWidth().clickable { nav.push(Route.Product(pid)) }.padding(vertical = 12.dp)
    Row(rowMod, verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    p.optString("name").ifBlank { "—" },
                    fontSize = 14.sp,
                    fontWeight = if (isCurrent) FontWeight.W600 else FontWeight.W500,
                    color = if (isCurrent) Palette.primary else Palette.ink,
                )
                if (isCurrent) {
                    Spacer(Modifier.width(6.dp))
                    CurrentTag()
                }
            }
            val desc = p.optString("description")
            if (desc.isNotEmpty()) Text(desc, fontSize = 12.sp, color = Palette.muted)
        }
        PricePill(p.optString("monthlyFee").ifBlank { "—" })
        Spacer(Modifier.width(4.dp))
        Icon(
            Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = "查看详情",
            tint = Palette.muted,
            modifier = Modifier.size(20.dp),
        )
    }
}

/** "当前套餐"实底蓝徽章:11sp 白字 + primary 实底 + 4dp 圆角,仅 CompareRow 用。 */
@Composable
private fun CurrentTag() {
    Text(
        "当前套餐",
        fontSize = 11.sp,
        color = Color.White,
        modifier = Modifier
            .background(Palette.primary, RoundedCornerShape(4.dp))
            .padding(horizontal = 6.dp, vertical = 2.dp),
    )
}

@Composable
private fun ContractCard() {
    AppCard {
        CardTitle("合约与说明")
        Notice("合约期内退订按未履约月份收取违约金;改套餐当月按新旧价按日折算(pro-rata)。")
    }
}

/** 底部固定 CTA:2dp tonalElevation 白底 Surface,内嵌通栏蓝色按钮。 */
@Composable
private fun BoxScope.CtaBar(product: JSONObject?, onClick: () -> Unit) {
    val fee = product?.optString("monthlyFee")?.takeIf { it.isNotBlank() } ?: "—"
    Surface(
        tonalElevation = 2.dp,
        color = Color.White,
        modifier = Modifier.fillMaxWidth().align(Alignment.BottomCenter),
    ) {
        Button(
            onClick = onClick,
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            shape = RoundedCornerShape(8.dp),
            modifier = Modifier.fillMaxWidth().padding(14.dp).height(44.dp),
        ) {
            Text("立即办理 ¥$fee/月", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Color.White)
        }
    }
}