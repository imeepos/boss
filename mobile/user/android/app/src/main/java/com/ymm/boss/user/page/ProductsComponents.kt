package com.ymm.boss.user.page

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.CardGiftcard
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.SignalCellularAlt
import androidx.compose.material.icons.filled.Speed
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.statusBarSolid
import org.json.JSONObject

// 服务 tab 产品页组件(ProductsScreen 拆分,≤300 行红线)。

/** 首屏骨架屏:与 ProductCard 同构的灰色占位卡(脉冲呼吸),数据到达即消失。 */
@Composable
internal fun SkeletonProducts() {
    val alpha by rememberInfiniteTransition(label = "skeleton").animateFloat(
        initialValue = 0.40f, targetValue = 0.80f,
        animationSpec = infiniteRepeatable(tween(durationMillis = 700), RepeatMode.Reverse),
        label = "skeleton-alpha",
    )
    Column(Modifier.fillMaxWidth()) {
        repeat(3) {
            AppCard {
                Column(
                    Modifier.fillMaxWidth().graphicsLayer { this.alpha = alpha },
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            Modifier.weight(1f).height(15.dp)
                                .background(Palette.line, RoundedCornerShape(4.dp)),
                        )
                        Spacer(Modifier.width(10.dp))
                        Box(Modifier.width(60.dp).height(20.dp).background(Palette.line, RoundedCornerShape(4.dp)))
                    }
                    Spacer(Modifier.height(12.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(Modifier.size(44.dp).background(Palette.line, RoundedCornerShape(10.dp)))
                        Spacer(Modifier.width(10.dp))
                        Column(Modifier.weight(1f)) {
                            Box(
                                Modifier.fillMaxWidth().height(12.dp)
                                    .background(Palette.line, RoundedCornerShape(4.dp)),
                            )
                            Spacer(Modifier.height(8.dp))
                            Box(
                                Modifier.fillMaxWidth(0.6f).height(12.dp)
                                    .background(Palette.line, RoundedCornerShape(4.dp)),
                            )
                        }
                    }
                }
            }
        }
    }
}

/** 服务页 Header:无标题,搜索框直接嵌入 48dp 状态栏色顶栏(半透明白胶囊)。 */
@Composable
internal fun ServiceSearchHeader(value: String, onChange: (String) -> Unit) {
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
internal fun CategorySeg(current: String, onSelect: (String) -> Unit) {
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
internal fun ProductCard(p: JSONObject, nav: Nav) {
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
internal fun PricePill(fee: String, recommended: Boolean) {
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
internal fun MetaLine(p: JSONObject) {
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
internal fun FeatureLine(icon: androidx.compose.ui.graphics.vector.ImageVector, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 3.dp)) {
        Icon(icon, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(14.dp))
        Spacer(Modifier.width(6.dp))
        Text(text, fontSize = 12.sp, color = Palette.muted)
    }
}

@Composable
internal fun AddonCard(addons: List<JSONObject>, nav: Nav) {
    AppCard {
        Column(Modifier.fillMaxWidth()) {
            CardTitle("增值服务", more = "进入管理 >") { nav.push(Route.Addon) }
            if (addons.isEmpty()) EmptyState("暂无可订购增值服务")
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