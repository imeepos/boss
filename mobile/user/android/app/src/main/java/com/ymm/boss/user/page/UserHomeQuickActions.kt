package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material.icons.outlined.HeadsetMic
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Router
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material.icons.outlined.TrendingUp
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.actionOrange
import com.ymm.boss.user.ui.actionPurple
import com.ymm.boss.user.ui.brandBlue
import com.ymm.boss.user.ui.successGreen

private data class QuickAction(val label: String, val icon: ImageVector, val tint: Color, val route: Route)

@Composable
internal fun QuickActions(onAction: (Route) -> Unit) {
    val actions = rememberQuickActions()
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 4.dp, bottom = 4.dp)
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

@Composable
private fun rememberQuickActions(): List<QuickAction> {
    val blue = brandBlue()
    val green = successGreen()
    val orange = actionOrange()
    val purple = actionPurple()
    return listOf(
        QuickAction("办套餐", Icons.Outlined.ShoppingBag, blue, Route.Products),
        QuickAction("查订单", Icons.Outlined.Assignment, green, Route.Orders),
        QuickAction("缴费", Icons.Outlined.Payments, orange, Route.Pay),
        QuickAction("报故障", Icons.Outlined.Build, purple, Route.Fault),
        QuickAction("充值", Icons.Outlined.TrendingUp, blue, Route.Topup),
        QuickAction("查用量", Icons.Outlined.Router, green, Route.Usage),
        QuickAction("消息", Icons.Outlined.Notifications, orange, Route.Messages),
        QuickAction("客服", Icons.Outlined.HeadsetMic, purple, Route.Complaint),
    )
}

@Composable
private fun GridItem(action: QuickAction, modifier: Modifier = Modifier, onAction: (Route) -> Unit) {
    Column(
        modifier = modifier
            .heightIn(min = 60.dp)
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = { onAction(action.route) })
            .padding(horizontal = 4.dp, vertical = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(action.icon, contentDescription = action.label, tint = action.tint, modifier = Modifier.size(24.dp))
        Text(
            action.label, fontSize = 12.sp, lineHeight = 14.sp, fontWeight = FontWeight.Normal,
            color = MaterialTheme.colorScheme.onSurface,
            modifier = Modifier.fillMaxWidth().padding(top = 6.dp),
            textAlign = TextAlign.Center,
            maxLines = 1,
        )
    }
}