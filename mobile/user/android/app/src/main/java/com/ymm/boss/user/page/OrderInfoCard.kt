package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.Palette
import kotlinx.coroutines.launch
import org.json.JSONObject

// 订单信息卡:产品/地址/下单时间/装维师傅(带呼叫入口)。

@Composable
internal fun OrderInfoCard(order: JSONObject?, wrapper: JSONObject?) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    AppCard {
        CardTitle("订单信息")
        Spacer(Modifier.height(4.dp))
        InfoRow("产品名称", order?.optString("productName").orDefault("—"))
        InfoRow("安装地址", order?.optString("address").orDefault("—"), icon = Icons.Filled.LocationOn)
        // submitedAt/technicianName/technicianPhoneMasked 在 OrderDetail 顶层,不在 inner order。
        InfoRow("下单时间", wrapper?.optString("submitedAt").orDefault("—"))
        val name = wrapper?.optString("technicianName").orDefault("")
        val phone = wrapper?.optString("technicianPhoneMasked").orDefault("")
        val techLine = when {
            name.isBlank() -> "尚未分配"
            phone.isBlank() -> name
            else -> "$name · $phone"
        }
        InfoRow(
            label = "装维师傅",
            value = techLine,
            trailing = if (name.isNotBlank() && order != null) {
                {
                    val orderNo = order.optString("orderNo")
                    Icon(
                        Icons.Filled.Call, contentDescription = "呼叫师傅",
                        tint = Palette.primary, modifier = Modifier.size(18.dp)
                            .clickable { scope.launch { dialTechnician(ctx, orderNo) } },
                    )
                }
            } else null,
        )
    }
}

@Composable
private fun InfoRow(label: String, value: String, icon: ImageVector? = null, trailing: (@Composable () -> Unit)? = null) {
    Row(
        Modifier.fillMaxWidth().height(IntrinsicSize.Min).padding(vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, fontSize = 13.sp, color = Palette.muted)
        Spacer(Modifier.width(16.dp))
        if (icon != null) {
            Icon(icon, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(14.dp))
            Spacer(Modifier.width(4.dp))
        }
        Text(value, fontSize = 13.sp, color = Palette.ink, modifier = Modifier.weight(1f))
        trailing?.invoke()
    }
}
