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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Router
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.auxText
import com.ymm.boss.user.ui.brandBlue
import com.ymm.boss.user.ui.homeHeaderGradient
import com.ymm.boss.user.ui.theme.Green500
import com.ymm.boss.user.ui.theme.OnGradient
import java.util.Calendar

internal const val HeaderHeightDp = 199

internal const val CardOverlapDp = 32

@Composable
internal fun HomeHeader(state: HomeUiState, onOpenMessages: () -> Unit) {
    val hour = remember { Calendar.getInstance().get(Calendar.HOUR_OF_DAY) }
    // 内容层:填满 PinnedGradientPage 头部槽位(高度由骨架统一),背景渐变由骨架底层负责
    Box(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp, vertical = 24.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    "${greetingFor(hour)}，${state.customerName.ifEmpty { "客户" }}",
                    fontSize = 22.sp, lineHeight = 28.sp, fontWeight = FontWeight.Bold,
                    color = OnGradient,
                    maxLines = 1,
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    state.phoneMasked.ifEmpty { "未绑定手机号" },
                    fontSize = 16.sp, lineHeight = 20.sp, color = OnGradient,
                )
                Spacer(modifier = Modifier.height(8.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(modifier = Modifier.size(8.dp).background(Green500, CircleShape))
                    Spacer(modifier = Modifier.width(6.dp))
                    Text(
                        state.onlineStatus.ifEmpty { "服务在线·网络正常" },
                        fontSize = 14.sp, lineHeight = 16.sp, color = OnGradient,
                    )
                }
            }
            Box {
                Icon(
                    Icons.Outlined.Notifications, contentDescription = "消息通知",
                    tint = OnGradient, modifier = Modifier
                        .sizeIn(minWidth = 48.dp, minHeight = 48.dp)
                        .clip(CircleShape)
                        .clickable(onClick = onOpenMessages)
                        .padding(12.dp),
                )
                if (state.hasUnread) {
                    Box(
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .padding(top = 8.dp, end = 8.dp)
                            .size(8.dp)
                            .background(Color(0xFFFF5252), CircleShape),
                    )
                }
            }
        }
    }
}

@Composable
internal fun BroadbandCard(state: HomeUiState, onOpen: () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 0.dp, bottom = 4.dp), // 顶:贴齐滚动区,圆角由区域裁剪呈现
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
        onClick = onOpen,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Outlined.Home, contentDescription = null, tint = brandBlue(), modifier = Modifier.size(24.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        "家庭宽带 ${state.planName.ifEmpty { "--" }}",
                        fontSize = 20.sp, lineHeight = 24.sp, fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurface,
                        maxLines = 1,
                    )
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                SubInfo(label = "本月账单", value = state.currentBill, modifier = Modifier.weight(1f))
                SubInfo(label = "套餐余额", value = state.balance, modifier = Modifier.weight(1f))
                SubInfo(label = "合约到期", value = state.contractEnd.ifEmpty { "--" }, modifier = Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun SubInfo(label: String, value: String, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier.padding(vertical = 6.dp, horizontal = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, fontSize = 13.sp, lineHeight = 16.sp, fontWeight = FontWeight.Normal, color = auxText())
        Spacer(modifier = Modifier.height(6.dp))
        Text(
            value, fontSize = 20.sp, lineHeight = 24.sp, fontWeight = FontWeight.Bold,
            color = brandBlue(),
            maxLines = 1,
            softWrap = false,
        )
    }
}

@Composable
private fun OnlineTag(label: String = "在网", active: Boolean = true) {
    Box(
        modifier = Modifier
            .heightIn(min = 24.dp)
            .background(
                if (active) Green500 else MaterialTheme.colorScheme.surfaceVariant,
                RoundedCornerShape(8.dp),
            )
            .padding(horizontal = 10.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            label, fontSize = 12.sp, lineHeight = 12.sp, fontWeight = FontWeight.Medium,
            color = if (active) Color.White else MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 1,
        )
    }
}

/** 在线判定:onlineStatus 含"在线"且不含异常字样视为正常;空串按默认在线处理。 */
internal fun isOnline(onlineStatus: String): Boolean {
    if (onlineStatus.isEmpty()) return true
    if (!onlineStatus.contains("在线")) return false
    return !onlineStatus.contains("异常") && !onlineStatus.contains("故障")
}

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
                        val label = service.statusLabel.ifEmpty { "在网" }
                        OnlineTag(label = label, active = label == "在网")
                    }
                }
            }
        }
    }
}