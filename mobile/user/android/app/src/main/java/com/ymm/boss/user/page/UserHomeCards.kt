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
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.auxText
import com.ymm.boss.user.ui.brandBlue
import com.ymm.boss.user.ui.homeHeaderGradient
import com.ymm.boss.user.ui.successGreen
import com.ymm.boss.user.ui.theme.Green500
import com.ymm.boss.user.ui.theme.OnGradient
import java.util.Calendar

internal const val HeaderHeightDp = 180

internal const val CardOverlapDp = 32

@Composable
internal fun HomeHeader(state: HomeUiState, onOpenMessages: () -> Unit) {
    val hour = remember { Calendar.getInstance().get(Calendar.HOUR_OF_DAY) }
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(HeaderHeightDp.dp)
            .background(homeHeaderGradient()),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .statusBarsPadding()
                .padding(horizontal = 24.dp, vertical = 24.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    "${greetingFor(hour)}，${state.customerName.ifEmpty { "客户" }}",
                    fontSize = 32.sp, lineHeight = 40.sp, fontWeight = FontWeight.Bold,
                    color = OnGradient,
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
                        "服务在线·${state.onlineStatus.ifEmpty { "网络正常" }}",
                        fontSize = 14.sp, lineHeight = 16.sp, color = OnGradient,
                    )
                }
            }
            Icon(
                Icons.Outlined.Notifications, contentDescription = "消息通知",
                tint = OnGradient, modifier = Modifier
                    .sizeIn(minWidth = 48.dp, minHeight = 48.dp)
                    .clip(CircleShape)
                    .clickable(onClick = onOpenMessages)
                    .padding(12.dp),
            )
        }
    }
}

@Composable
internal fun WelcomeCard(state: HomeUiState) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(modifier = Modifier.size(10.dp).background(successGreen(), CircleShape))
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                state.onlineStatus.ifEmpty { "网络正常" },
                fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.onSurface,
            )
        }
    }
}

@Composable
internal fun BroadbandCard(state: HomeUiState, onOpen: () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
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
                    Icon(Icons.Outlined.Router, contentDescription = null, tint = brandBlue(), modifier = Modifier.size(24.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        "家庭宽带 ${state.planName.ifEmpty { "--" }}",
                        fontSize = 20.sp, lineHeight = 24.sp, fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurface,
                    )
                }
                OnlineTag()
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
        Text(label, fontSize = 14.sp, lineHeight = 16.sp, fontWeight = FontWeight.Regular, color = auxText())
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            value, fontSize = 32.sp, lineHeight = 40.sp, fontWeight = FontWeight.Bold,
            color = brandBlue(),
        )
    }
}

@Composable
private fun OnlineTag() {
    Text(
        "在网", fontSize = 14.sp, lineHeight = 16.sp, fontWeight = FontWeight.Medium,
        color = Color.White,
        modifier = Modifier
            .background(Green500, RoundedCornerShape(8.dp))
            .padding(horizontal = 10.dp, vertical = 3.dp)
            .heightIn(min = 24.dp),
    )
}

@Composable
internal fun OrderCardShell(onOpenAll: () -> Unit, content: @Composable () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("进行中订单", fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold, color = brandBlue())
                Text(
                    "全部 >", fontSize = 14.sp, lineHeight = 16.sp, color = auxText(),
                    modifier = Modifier
                        .sizeIn(minWidth = 48.dp, minHeight = 48.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .clickable(onClick = onOpenAll)
                        .padding(horizontal = 8.dp, vertical = 12.dp),
                )
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
            .padding(top = 4.dp)
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
internal fun MyServiceCard(service: HomeService?, onClick: () -> Unit) {
    val cardIcon: ImageVector = Icons.Outlined.Router
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.5.dp),
        onClick = onClick,
    ) {
        if (service == null) {
            Text(
                "暂无在用服务", fontSize = 14.sp, color = auxText(),
                modifier = Modifier.padding(16.dp),
            )
        } else {
            Row(
                modifier = Modifier.fillMaxWidth().padding(16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    cardIcon, contentDescription = null, tint = brandBlue(),
                    modifier = Modifier.size(40.dp).background(MaterialTheme.colorScheme.surfaceVariant, CircleShape).padding(8.dp),
                )
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text("我的服务 · ${service.name}", fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.onSurface)
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        service.desc.ifEmpty { "查看套餐详情" }, fontSize = 14.sp, lineHeight = 16.sp,
                        color = auxText(),
                    )
                }
                Spacer(modifier = Modifier.width(8.dp))
                OnlineTag()
            }
        }
    }
}