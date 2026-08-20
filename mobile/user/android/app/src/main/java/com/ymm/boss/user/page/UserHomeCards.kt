package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Assignment
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Router
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.brandBlue
import java.util.Calendar

/** 用户首页卡片组件,规格见设计稿 E/I 节;颜色一律 colorScheme/色板常量。 */

@Composable
internal fun GreetingCard(state: HomeUiState) {
    val hour = remember { Calendar.getInstance().get(Calendar.HOUR_OF_DAY) }
    Column(
        modifier = Modifier
            .fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)
            .heightIn(min = 120.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(16.dp))
            .padding(16.dp),
        verticalArrangement = Arrangement.SpaceBetween,
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Column {
                Text(
                    "${greetingFor(hour)}，${state.customerName.ifEmpty { "客户" }}",
                    fontSize = 20.sp, lineHeight = 28.sp, fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    state.phoneMasked.ifEmpty { "未绑定手机号" },
                    fontSize = 14.sp, lineHeight = 20.sp,
                    color = MaterialTheme.colorScheme.tertiary,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(modifier = Modifier.size(8.dp).background(MaterialTheme.colorScheme.primary, CircleShape))
            Text(
                "  ${state.onlineStatus}",
                fontSize = 14.sp, lineHeight = 20.sp, color = MaterialTheme.colorScheme.tertiary,
            )
        }
    }
}

private data class QuickAction(val label: String, val icon: ImageVector, val route: Route)

@Composable
internal fun QuickActions(onAction: (Route) -> Unit) {
    val actions = listOf(
        QuickAction("办套餐", Icons.Outlined.ShoppingBag, Route.Products),
        QuickAction("查订单", Icons.Outlined.Assignment, Route.Orders),
        QuickAction("缴费用", Icons.Outlined.Payments, Route.Pay),
        QuickAction("报故障", Icons.Outlined.Build, Route.Fault),
    )
    Row(
        modifier = Modifier
            .fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)
            .heightIn(min = 80.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(12.dp))
            .padding(16.dp),
    ) {
        actions.forEach { a ->
            Column(
                modifier = Modifier
                    .weight(1f).fillMaxHeight().sizeIn(minWidth = 48.dp, minHeight = 48.dp)
                    .clickable(onClick = { onAction(a.route) }),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                Icon(a.icon, contentDescription = a.label, tint = brandBlue(), modifier = Modifier.size(28.dp))
                Text(a.label, fontSize = 12.sp, lineHeight = 20.sp, color = MaterialTheme.colorScheme.onSurface, modifier = Modifier.padding(top = 4.dp))
            }
        }
    }
}

@Composable
internal fun OrderCard(order: HomeOrder, onOpen: (String) -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp)
            .heightIn(min = 120.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(12.dp))
            .clickable(onClick = { onOpen(order.orderNo) })
            .padding(16.dp),
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(order.orderNo, fontSize = 16.sp, lineHeight = 24.sp, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurface)
            StatusTag(text = order.statusLabel, done = order.status == "DONE")
        }
        Text(
            order.productName, fontSize = 16.sp, lineHeight = 24.sp,
            color = MaterialTheme.colorScheme.onSurface, modifier = Modifier.padding(top = 6.dp),
        )
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 2.dp)) {
            Icon(Icons.Outlined.LocationOn, contentDescription = null, tint = MaterialTheme.colorScheme.tertiary, modifier = Modifier.size(16.dp))
            Text(
                " ${order.address}", fontSize = 14.sp, lineHeight = 20.sp,
                color = MaterialTheme.colorScheme.tertiary,
            )
        }
        InstallProgress(stage = order.stage, modifier = Modifier.padding(top = 10.dp))
    }
}

@Composable
private fun InstallProgress(stage: Int, modifier: Modifier = Modifier) {
    Row(modifier = modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text("上门安装", fontSize = 14.sp, lineHeight = 20.sp, color = brandBlue())
        Text(
            "  ${stage}/12", fontSize = 12.sp, lineHeight = 20.sp,
            color = MaterialTheme.colorScheme.tertiary,
        )
        Spacer(modifier = Modifier.width(10.dp))
        LinearProgressIndicator(
            progress = { (stage.coerceIn(0, 12)) / 12f },
            modifier = Modifier.width(200.dp),
            color = brandBlue(),
            trackColor = MaterialTheme.colorScheme.surfaceVariant,
        )
    }
}

@Composable
internal fun MyServiceCard(service: HomeService?, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp)
            .heightIn(min = 80.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (service == null) {
            Text("暂无在用服务", fontSize = 14.sp, color = MaterialTheme.colorScheme.tertiary)
        } else {
            Icon(
                Icons.Outlined.Router, contentDescription = null, tint = brandBlue(),
                modifier = Modifier.size(40.dp).background(MaterialTheme.colorScheme.surfaceVariant, CircleShape).padding(8.dp),
            )
            Column(modifier = Modifier.padding(start = 12.dp).weight(1f)) {
                Text(service.name, fontSize = 16.sp, lineHeight = 24.sp, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurface)
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 2.dp)) {
                    Box(modifier = Modifier.size(8.dp).background(MaterialTheme.colorScheme.primary, CircleShape))
                    Text("  ${service.desc}", fontSize = 14.sp, lineHeight = 20.sp, color = MaterialTheme.colorScheme.tertiary)
                }
            }
            StatusTag(text = service.statusLabel, done = true)
        }
    }
}

@Composable
private fun StatusTag(text: String, done: Boolean) {
    val color = if (done) MaterialTheme.colorScheme.primary else brandBlue()
    Text(
        text, fontSize = 12.sp, color = color,
        modifier = Modifier.background(color.copy(alpha = 0.12f), RoundedCornerShape(6.dp)).padding(horizontal = 8.dp, vertical = 2.dp),
    )
}
