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
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.auxText
import com.ymm.boss.user.ui.brandBlue
import com.ymm.boss.user.ui.homeHeaderGradient
import com.ymm.boss.user.ui.successGreen
import com.ymm.boss.user.ui.theme.OnGradient
import java.util.Calendar

/** 用户首页卡片组件,规格见设计稿 D+/E/I 节;颜色一律 colorScheme/色板常量。 */

/** 渐变头部高度(含状态栏区域)。 */
internal const val HeaderHeightDp = 220

/** 首卡片与渐变底部的视觉重合量(设计稿量得约 30dp,取 4dp 栅格 32dp)。 */
internal const val CardOverlapDp = 32

/** 顶部蓝色渐变标题区:约 45° 渐变,延伸到状态栏后方,高约 220dp(含状态栏)。 */
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
                .padding(horizontal = 16.dp, vertical = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    "${greetingFor(hour)}，${state.customerName.ifEmpty { "客户" }}",
                    fontSize = 24.sp, lineHeight = 32.sp, fontWeight = FontWeight.Bold,
                    color = OnGradient,
                )
                Text(
                    state.phoneMasked.ifEmpty { "未绑定手机号" },
                    fontSize = 16.sp, lineHeight = 22.sp, color = OnGradient,
                    modifier = Modifier.padding(top = 4.dp),
                )
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

/** 欢迎卡片:圆角 20dp,绿色点 + 服务状态文案。 */
@Composable
internal fun WelcomeCard(state: HomeUiState) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 16.dp, end = 16.dp)
            .heightIn(min = 80.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(20.dp))
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(modifier = Modifier.size(10.dp).background(successGreen(), CircleShape))
        Text(
            "  ${state.onlineStatus.ifEmpty { "网络正常" }}",
            fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Medium,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** 家庭宽带卡片:标题+绿色在网标签,三个子信息卡片(本月账单/套餐余额/合约到期)。 */
@Composable
internal fun BroadbandCard(state: HomeUiState, onOpen: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(16.dp))
            .clickable(onClick = onOpen)
            .padding(16.dp),
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.Router, contentDescription = null, tint = brandBlue(), modifier = Modifier.size(24.dp))
                Text(
                    " 家庭宽带 ${state.planName.ifEmpty { "--" }}",
                    fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Bold,
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

/** 子信息项:设计稿无独立底色(随卡面),标签辅助灰、数值品牌蓝粗体。 */
@Composable
private fun SubInfo(label: String, value: String, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier.padding(vertical = 6.dp, horizontal = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium, color = auxText())
        Text(
            value, fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Bold,
            color = brandBlue(), modifier = Modifier.padding(top = 2.dp),
        )
    }
}

/** 绿色"在网"标签。 */
@Composable
private fun OnlineTag() {
    Text(
        "在网", fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium, color = successGreen(),
        modifier = Modifier
            .background(successGreen().copy(alpha = 0.12f), RoundedCornerShape(8.dp))
            .padding(horizontal = 8.dp, vertical = 2.dp),
    )
}

/** 进行中订单卡片外壳:标题蓝色 + "全部 >"链接。 */
@Composable
internal fun OrderCardShell(onOpenAll: () -> Unit, content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(16.dp))
            .padding(16.dp),
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text("进行中订单", fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Bold, color = brandBlue())
            Text(
                "全部 ›", fontSize = 14.sp, lineHeight = 20.sp, color = auxText(),
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

/** 单条订单:订单号 + 地址 + 进度条(stage/12)。 */
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
        Text(order.orderNo, fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurface)
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 2.dp)) {
            Icon(Icons.Outlined.LocationOn, contentDescription = null, tint = auxText(), modifier = Modifier.size(16.dp))
            Text(
                " ${order.address}", fontSize = 14.sp, lineHeight = 20.sp, color = auxText(),
                maxLines = 1,
            )
        }
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(top = 10.dp),
        ) {
            Text(
                order.statusLabel, fontSize = 14.sp, lineHeight = 20.sp,
                fontWeight = FontWeight.Medium, color = brandBlue(),
            )
            Text(
                "  ${order.stage}/12", fontSize = 14.sp, lineHeight = 20.sp, color = auxText(),
            )
            Spacer(modifier = Modifier.width(8.dp))
            LinearProgressIndicator(
                progress = { order.stage.coerceIn(0, 12) / 12f },
                modifier = Modifier.weight(1f).height(4.dp).clip(RoundedCornerShape(2.dp)),
                color = brandBlue(),
                trackColor = MaterialTheme.colorScheme.surfaceVariant,
            )
        }
    }
}

/** 我的服务卡片:图标 + 名称/描述 + 在网标签。 */
@Composable
internal fun MyServiceCard(service: HomeService?, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .heightIn(min = 90.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (service == null) {
            Text("暂无在用服务", fontSize = 14.sp, color = auxText())
        } else {
            Icon(
                Icons.Outlined.Router, contentDescription = null, tint = brandBlue(),
                modifier = Modifier.size(40.dp).background(MaterialTheme.colorScheme.surfaceVariant, CircleShape).padding(8.dp),
            )
            Column(modifier = Modifier.padding(start = 12.dp).weight(1f).fillMaxHeight()) {
                Text("我的服务 · ${service.name}", fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.onSurface)
                Text(
                    service.desc.ifEmpty { "查看套餐详情" }, fontSize = 14.sp, lineHeight = 20.sp,
                    color = auxText(), modifier = Modifier.padding(top = 2.dp),
                )
            }
            OnlineTag()
        }
    }
}
