package com.ymm.boss.user.page

import androidx.compose.foundation.background
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
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

private const val DEMO_ADDRESS_ID = "ADDR-001"

// 对应草稿 docs/user/product.html:套餐详情?id=,specs/对比 + 立即办理。
// 视觉基准与 ProductsPage 对齐:价格右上角胶囊、卡片标题 15sp W600、金额 Bold。
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
        } catch (e: Exception) { err = "套餐详情加载失败" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("套餐详情", onBack = { nav.pop() })
        if (err.isNotEmpty()) Notice(err, Palette.err)
        DetailCard(product, specs)
        CompareCard(compare, nav)
        ContractCard()
        CtaBar(nav, id, product)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun DetailCard(product: JSONObject?, specs: List<JSONObject>) {
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                product?.optString("name") ?: "加载中…",
                fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(10.dp))
            DetailPricePill(product?.optString("monthlyFee") ?: "—")
        }
        Notice(product?.optString("description") ?: "—")
        specs.forEach { s ->
            CellRow(title = s.optString("label"), right = {
                Text(s.optString("value"), fontSize = 13.sp, color = Palette.muted)
            })
        }
    }
}

@Composable
private fun CompareCard(compare: List<JSONObject>, nav: Nav) {
    AppCard {
        CardTitle("套餐对比", more = "同档可选")
        if (compare.isEmpty()) EmptyState("暂无可比套餐")
        compare.forEach { p ->
            CellRow(
                title = p.optString("name"), desc = p.optString("description"),
                onClick = { nav.push(Route.Product(p.optString("productId"))) },
                right = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        DetailPricePill(p.optString("monthlyFee"))
                        Spacer(Modifier.width(4.dp))
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = "查看详情",
                            tint = Palette.muted,
                            modifier = Modifier.size(20.dp),
                        )
                    }
                },
            )
        }
    }
}

@Composable
private fun ContractCard() {
    AppCard {
        CardTitle("合约与说明")
        Notice("合约期内退订按未履约月份收取违约金;改套餐当月按新旧价按日折算(pro-rata)。")
    }
}

@Composable
private fun CtaBar(nav: Nav, id: String, product: JSONObject?) {
    val scope = rememberCoroutineScope()
    Button(
        onClick = {
            scope.launch {
                try {
                    val o = OrderApi.submit(id, DEMO_ADDRESS_ID)
                    nav.push(Route.Order(o.optString("orderNo")))
                } catch (e: Exception) { } // 下单失败静默,与草稿 catch 行为一致
            }
        },
        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp).height(44.dp),
    ) {
        Text(
            "立即办理 ¥${product?.optString("monthlyFee") ?: "—"}/月",
            fontSize = 14.sp, fontWeight = FontWeight.W500,
        )
    }
}

/** 与 ProductsPage 的 PricePill 同形态(该处为 private 无法直接引用):普通档品牌蓝浅底。 */
@Composable
private fun DetailPricePill(fee: String) {
    Row(
        verticalAlignment = Alignment.Bottom,
        modifier = Modifier
            .background(Palette.primary.copy(alpha = 0.12f), RoundedCornerShape(8.dp))
            .padding(horizontal = 10.dp, vertical = 4.dp),
    ) {
        Text("¥", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = Palette.primary)
        Text(fee, fontSize = 16.sp, fontWeight = FontWeight.Bold, color = Palette.primary)
        Text("/月", fontSize = 10.sp, color = Palette.primary, modifier = Modifier.padding(bottom = 1.dp))
    }
}
