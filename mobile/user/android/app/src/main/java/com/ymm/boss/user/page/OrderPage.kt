package com.ymm.boss.user.page

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.HourglassEmpty
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应 design/order-detail-v1.png:状态头 + 信息卡 + 4 里程碑 + 预计上门条 + 12 环节时间线 + 动作栏。
@Composable
fun OrderScreen(nav: Nav, no: String) {
    var detail by remember { mutableStateOf<JSONObject?>(null) }
    var timeline by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    var showCancel by remember { mutableStateOf(false) }
    var notFound by remember { mutableStateOf(false) }
    LaunchedEffect(no, nav.refreshTick) {
        try {
            val resp = OrderApi.detail(no)
            detail = resp.optJSONObject("order")
            timeline = resp.optJSONArray("timeline").toObjList()
            notFound = false
            err = ""
        } catch (e: Api.HttpError) {
            if (e.status == 404) notFound = true else err = "订单详情加载失败,请稍后重试"
        } catch (e: Exception) {
            err = "订单详情加载失败,请稍后重试"
        }
    }

    if (notFound) {
        Column(Modifier.fillMaxSize()) {
            TopBar("订单详情", onBack = { nav.pop() })
            EmptyState("订单不存在或已删除")
        }
        return
    }

    val order = detail
    val status = order?.optString("status") ?: ""
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(
            title = "订单详情",
            onBack = { nav.pop() },
            action = if (status in setOf("PENDING", "RESERVED", "INSTALLING")) "取消" else null,
            onAction = { showCancel = true },
        )
        if (err.isNotEmpty()) Notice(err, Palette.err)
        StatusHeader(order)
        InfoCard(order)
        MilestoneBlock(order)
        if (status == "INSTALLING") EstimateBanner(order)
        TimelineCard(detail, timeline)
        ActionBar(nav, no, order)
        Spacer(Modifier.height(12.dp))
    }
    if (showCancel) CancelDialog(nav, no) { showCancel = false }
}

@Composable
private fun StatusHeader(order: JSONObject?) {
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconTile(statusIcon(order), statusTint(order), size = 48.dp, corner = 12.dp)
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    order?.optString("statusLabel").orDefault(order?.optString("status").labelOf()),
                    fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Palette.ink,
                )
                val desc = order?.optString("statusDesc").orEmpty()
                if (desc.isNotBlank()) {
                    Text(desc, fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 4.dp))
                }
            }
            Spacer(Modifier.width(12.dp))
            Column(horizontalAlignment = Alignment.End) {
                Text("订单号", fontSize = 11.sp, color = Palette.subtle)
                Text(
                    order?.optString("orderNo").orDefault("—"),
                    fontSize = 13.sp, fontWeight = FontWeight.W600, color = Palette.ink,
                    modifier = Modifier.padding(top = 2.dp),
                )
            }
        }
    }
}

private fun statusIcon(order: JSONObject?): ImageVector = when (order?.optString("status")) {
    "DONE" -> Icons.Filled.CheckCircle
    "INSTALLING" -> Icons.Filled.Build
    "RESERVED" -> Icons.Filled.HourglassEmpty
    else -> Icons.Filled.HourglassEmpty
}

private fun statusTint(order: JSONObject?): androidx.compose.ui.graphics.Color = when (order?.optString("status")) {
    "DONE" -> Palette.success
    "INSTALLING" -> Palette.primary
    "RESERVED" -> Palette.warn
    else -> Palette.muted
}

private fun String?.labelOf(): String = when (this) {
    "PENDING" -> "待核查"
    "RESERVED" -> "已预占"
    "INSTALLING" -> "装维中"
    "DONE" -> "已完成"
    "CANCELLED" -> "已取消"
    else -> "—"
}

private fun String?.orDefault(fallback: String): String =
    if (this.isNullOrBlank()) fallback else this

@Composable
private fun InfoCard(order: JSONObject?) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    AppCard {
        CardTitle("订单信息")
        Spacer(Modifier.height(4.dp))
        InfoRow("产品名称", order?.optString("productName").orDefault("—"))
        InfoRow("安装地址", order?.optString("address").orDefault("—"), icon = Icons.Filled.LocationOn)
        InfoRow("下单时间", order?.optString("submitedAt").orDefault("—"))
        val name = order?.optString("technicianName").orDefault("")
        val phone = order?.optString("technicianPhoneMasked").orDefault("")
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

@Composable
private fun MilestoneBlock(order: JSONObject?) {
    AppCard {
        CardTitle("进度概览", "${order?.optInt("completedStage") ?: 0}/12 已完成")
        Spacer(Modifier.height(8.dp))
        OrderStepper(milestoneOf(order?.optInt("stage") ?: 1))
    }
}

@Composable
private fun EstimateBanner(order: JSONObject?) {
    val estimate = order?.optString("estimateFinish").orEmpty()
    if (estimate.isBlank()) return
    AppCard(outer = PaddingValues(horizontal = 14.dp, vertical = 6.dp)) {
        Row(
            Modifier.fillMaxWidth().background(Palette.primary.copy(alpha = 0.08f), RoundedCornerShape(8.dp))
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Filled.Schedule, contentDescription = null, tint = Palette.primary, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Column(Modifier.weight(1f)) {
                Text("预计上门:$estimate", fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink)
                Text("如有变动,师傅将提前电话联系您", fontSize = 11.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 2.dp))
            }
            Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(18.dp))
        }
    }
}

@Composable
private fun TimelineCard(detail: JSONObject?, timeline: List<JSONObject>) {
    AppCard {
        CardTitle("装维进度", "共 12 个环节")
        if (timeline.isEmpty()) {
            Text("进度加载中…", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
            return@AppCard
        }
        timeline.forEachIndexed { i, t -> TimelineItem(t, isLast = i == timeline.size - 1) }
    }
}

@Composable
private fun TimelineItem(t: JSONObject, isLast: Boolean) {
    val result = t.optString("result")
    val nodeColor = when (result) { "DONE" -> Palette.success; "DOING" -> Palette.primary; else -> Palette.line }
    Row(Modifier.height(IntrinsicSize.Min).padding(vertical = 4.dp)) {
        Column(Modifier.width(14.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(Modifier.size(8.dp).background(nodeColor, CircleShape))
            if (!isLast) Box(Modifier.width(2.dp).fillMaxHeight().padding(horizontal = 3.dp).background(Palette.line))
        }
        Column(Modifier.padding(start = 8.dp, bottom = if (isLast) 0.dp else 12.dp).weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    "${t.optInt("stage")} ${t.optString("title")}",
                    fontSize = 13.5.sp,
                    color = if (result == "PENDING") Palette.muted else Palette.ink,
                    fontWeight = if (result == "DOING") FontWeight.Bold else FontWeight.Normal,
                    modifier = Modifier.weight(1f),
                )
                val ts = t.optString("finishedAt").orEmpty()
                if (ts.isNotBlank()) Text(ts, fontSize = 12.sp, color = Palette.muted)
            }
            val meta = t.optString("meta").orEmpty()
            if (meta.isNotBlank()) Text(meta, fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(top = 2.dp))
        }
    }
}

@Composable
private fun ActionBar(nav: Nav, no: String, order: JSONObject?) {
    val status = order?.optString("status") ?: return
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        when (status) {
            "PENDING", "RESERVED", "INSTALLING" -> {
                Button(
                    onClick = { scope.launch { urge(ctx, no, nav) } },
                    colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                    modifier = Modifier.weight(1f),
                ) { Text("催单", fontSize = 15.sp, fontWeight = FontWeight.W600, color = androidx.compose.ui.graphics.Color.White) }
                OutlinedButton(
                    onClick = { scope.launch { dialTechnician(ctx, no) } },
                    modifier = Modifier.weight(1f),
                ) { Text("联系师傅", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
                OutlinedButton(
                    onClick = { /* TODO: 跳转变更地址子页 */ },
                    modifier = Modifier.weight(1f),
                ) { Text("变更地址", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
            }
            "DONE" -> {
                if (order.optBoolean("canRate")) {
                    Button(
                        onClick = { nav.push(com.ymm.boss.user.ui.Route.Rate(no)) },
                        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                        modifier = Modifier.weight(1f),
                    ) { Text("去评价", fontSize = 15.sp, fontWeight = FontWeight.W600, color = androidx.compose.ui.graphics.Color.White) }
                } else {
                    Box(
                        Modifier.weight(1f).background(Palette.success.copy(alpha = 0.08f), RoundedCornerShape(8.dp)).padding(vertical = 12.dp),
                        contentAlignment = Alignment.Center,
                    ) { Text("订单已完成,感谢您的选择!", fontSize = 13.sp, color = Palette.success) }
                }
                OutlinedButton(
                    onClick = { scope.launch { dialTechnician(ctx, no) } },
                    modifier = Modifier.weight(1f),
                ) { Text("联系师傅", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
            }
            "CANCELLED" -> {
                Box(Modifier.weight(1f).padding(vertical = 4.dp), contentAlignment = Alignment.Center) {
                    Text("订单已取消", fontSize = 13.sp, color = Palette.muted)
                }
                OutlinedButton(onClick = { nav.push(com.ymm.boss.user.ui.Route.Complaint) }, modifier = Modifier.weight(1f)) {
                    Text("联系客服", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary)
                }
            }
        }
    }
}

private suspend fun urge(ctx: Context, no: String, nav: Nav) {
    try {
        OrderApi.urge(no)
        Toast.makeText(ctx, "已通知师傅加紧处理", Toast.LENGTH_SHORT).show()
    } catch (e: Exception) {
        Toast.makeText(ctx, "催单失败,请稍后重试", Toast.LENGTH_SHORT).show()
    }
}

// 装维师傅明文电话走 GET orders/{orderNo}/technician-contact,取到后拉起拨号盘。
private suspend fun dialTechnician(ctx: Context, no: String) {
    try {
        val phone = OrderApi.technicianContact(no).optString("phone")
        if (phone.isBlank()) throw Api.HttpError(404, "empty phone")
        ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
    } catch (e: Exception) {
        Toast.makeText(ctx, "暂无法获取师傅电话,请稍后重试", Toast.LENGTH_SHORT).show()
    }
}

@Composable
private fun CancelDialog(nav: Nav, no: String, onDismiss: () -> Unit) {
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("确认取消该订单?") },
        text = { Text("取消后将释放预占端口并关闭订单。") },
        confirmButton = {
            TextButton(onClick = {
                scope.launch {
                    try {
                        OrderApi.cancel(no)
                        Toast.makeText(ctx, "订单已取消", Toast.LENGTH_SHORT).show()
                        onDismiss()
                        nav.pop()
                    } catch (e: Exception) {
                        Toast.makeText(ctx, "取消失败,请稍后重试", Toast.LENGTH_SHORT).show()
                        onDismiss()
                    }
                }
            }) { Text("确认取消", color = Palette.err) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("再想想", color = Palette.muted) } },
    )
}