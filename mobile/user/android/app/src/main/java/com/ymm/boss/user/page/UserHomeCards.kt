package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.Router
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.auxText
import com.ymm.boss.user.ui.brandBlue

// 首页订单/增值服务卡;头部问候区与宽带卡拆见 UserHomeHeaderCard.kt。

@Composable
internal fun OrderCardShell(onOpenAll: () -> Unit, content: @Composable () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 4.dp, bottom = 4.dp),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
    ) {
        Column(modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 12.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("进行中订单", fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold, color = brandBlue())
                Box(
                    modifier = Modifier
                        .sizeIn(minWidth = 48.dp, minHeight = 36.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .clickable(onClick = onOpenAll),
                    contentAlignment = Alignment.Center,
                ) {
                    Text("全部 >", fontSize = 13.sp, lineHeight = 16.sp, color = auxText())
                }
            }
            content()
        }
    }
}

@Composable
internal fun OrderItem(order: HomeOrder, onOpen: (String) -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = { onOpen(order.orderNo) })
            .padding(vertical = 8.dp),
    ) {
        Text(order.orderNo, fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurface)
        Spacer(modifier = Modifier.height(2.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Outlined.LocationOn, contentDescription = null, tint = auxText(), modifier = Modifier.size(16.dp))
            Spacer(modifier = Modifier.width(2.dp))
            Text(
                order.address, fontSize = 14.sp, lineHeight = 16.sp, color = auxText(),
                maxLines = 1,
            )
        }
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(top = 10.dp),
        ) {
            Text(
                order.statusLabel, fontSize = 14.sp, lineHeight = 16.sp,
                fontWeight = FontWeight.Medium, color = brandBlue(),
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                "${order.stage}/12", fontSize = 14.sp, lineHeight = 16.sp, color = auxText(),
            )
            Spacer(modifier = Modifier.width(8.dp))
            LinearProgressIndicator(
                progress = { order.stage.coerceIn(0, 12) / 12f },
                modifier = Modifier.weight(1f).height(8.dp),
                color = brandBlue(),
                trackColor = MaterialTheme.colorScheme.surfaceVariant,
            )
        }
    }
}

@Composable
internal fun ActiveServicesCard(services: List<HomeService>, onClick: () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 4.dp, bottom = 8.dp),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("已生效增值服务", fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold, color = brandBlue())
            if (services.isEmpty()) {
                EmptyState("暂无已生效增值服务", modifier = Modifier.padding(top = 4.dp))
            } else {
                services.forEachIndexed { index, service ->
                    if (index > 0) {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 12.dp)
                                .height(1.dp)
                                .background(MaterialTheme.colorScheme.surfaceVariant),
                        )
                    }
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 12.dp)
                            .clip(RoundedCornerShape(8.dp))
                            .clickable(onClick = onClick),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            Icons.Outlined.Router, contentDescription = null, tint = brandBlue(),
                            modifier = Modifier
                                .size(36.dp)
                                .background(MaterialTheme.colorScheme.surfaceVariant, CircleShape)
                                .padding(8.dp),
                        )
                        Spacer(modifier = Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(service.name, fontSize = 15.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurface, maxLines = 1)
                            Spacer(modifier = Modifier.height(2.dp))
                            Text(
                                service.desc.ifEmpty { "查看套餐详情" }, fontSize = 13.sp, lineHeight = 16.sp,
                                color = auxText(), maxLines = 1,
                            )
                        }
                        Spacer(modifier = Modifier.width(8.dp))
                        val label = service.statusLabel.ifEmpty { "已生效" }
                        OnlineTag(label = label, active = service.status == "ACTIVE")
                    }
                }
            }
        }
    }
}