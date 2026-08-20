package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Assignment
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.CardGiftcard
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Receipt
import androidx.compose.material.icons.outlined.ReceiptLong
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.brandBlue
import com.ymm.boss.user.ui.successGreen

/** 快捷功能宫格(设计稿 D+/E 节):4×2,容器圆角 16dp,图标蓝/绿/橙/紫,间距 12dp。 */

private data class QuickAction(val label: String, val icon: ImageVector, val tint: Color, val route: Route)

@Composable
internal fun QuickActions(onAction: (Route) -> Unit) {
    val actions = listOf(
        QuickAction("办套餐", Icons.Outlined.ShoppingBag, brandBlue(), Route.Products),
        QuickAction("查订单", Icons.Outlined.Assignment, brandBlue(), Route.Orders),
        QuickAction("缴费用", Icons.Outlined.Payments, successGreen(), Route.Pay),
        QuickAction("报故障", Icons.Outlined.Build, Palette.orange, Route.Fault),
        QuickAction("查账单", Icons.Outlined.ReceiptLong, brandBlue(), Route.Bills),
        QuickAction("开发票", Icons.Outlined.Receipt, successGreen(), Route.Invoice),
        QuickAction("领优惠", Icons.Outlined.CardGiftcard, Palette.purple, Route.Coupon),
        QuickAction("消息", Icons.Outlined.Notifications, Palette.orange, Route.Messages),
    )
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(16.dp))
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        actions.chunked(4).forEach { rowActions ->
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                rowActions.forEach { a ->
                    GridItem(action = a, modifier = Modifier.weight(1f), onAction = onAction)
                }
            }
        }
    }
}

/** 单个宫格项:clip+clickable 自带涟漪,最小触达 48dp(验收 3)。 */
@Composable
private fun GridItem(action: QuickAction, modifier: Modifier = Modifier, onAction: (Route) -> Unit) {
    Column(
        modifier = modifier
            .heightIn(min = 48.dp)
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = { onAction(action.route) })
            .padding(vertical = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(action.icon, contentDescription = action.label, tint = action.tint, modifier = Modifier.size(28.dp))
        Text(
            action.label, fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium,
            color = MaterialTheme.colorScheme.onSurface, modifier = Modifier.padding(top = 4.dp),
        )
    }
}
