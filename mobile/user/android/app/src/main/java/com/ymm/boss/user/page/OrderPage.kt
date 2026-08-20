package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/order.html:订单详情?no=,12 环节进度、取消/催办/改地址入口。
@Composable
fun OrderScreen(nav: Nav, no: String) {
    var detail by remember { mutableStateOf<JSONObject?>(null) }
    var timeline by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var err by remember { mutableStateOf("") }
    var showCancel by remember { mutableStateOf(false) }
    LaunchedEffect(no, nav.refreshTick) {
        try {
            detail = OrderApi.detail(no)
            timeline = detail?.optJSONArray("timeline").toObjList()
        } catch (e: Exception) { err = "订单详情加载失败" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("订单详情", onBack = { nav.pop() }, action = "取消") { showCancel = true }
        if (err.isNotEmpty()) ErrText(err)
        SummaryCard(detail?.optJSONObject("order"), detail)
        TimelineCard(detail, timeline)
        ActionBar(nav, no)
        Spacer(Modifier.height(12.dp))
    }
    if (showCancel) CancelDialog(nav, no) { showCancel = false }
}

@Composable
private fun ErrText(err: String) {
    Text(err, fontSize = 12.5.sp, color = Palette.err, modifier = Modifier.padding(horizontal = 14.dp, vertical = 4.dp))
}

@Composable
private fun SummaryCard(order: JSONObject?, detail: JSONObject?) {
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(order?.optString("orderNo") ?: "加载中…", fontSize = 16.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
            Spacer(Modifier.weight(1f))
            Tag(order?.optString("statusLabel").takeUnless { it.isNullOrEmpty() } ?: "—", Palette.primary)
        }
        InfoRow("产品", order?.optString("productName").takeUnless { it.isNullOrEmpty() } ?: "—")
        InfoRow("安装地址", order?.optString("address").takeUnless { it.isNullOrEmpty() } ?: "—")
        InfoRow("下单时间", detail?.optString("submitedAt").takeUnless { it.isNullOrEmpty() } ?: "—")
        InfoRow("装维师傅", techLine(detail))
    }
}

private fun techLine(detail: JSONObject?): String {
    val name = detail?.optString("technicianName").takeUnless { it.isNullOrEmpty() } ?: return "—"
    val phone = detail?.optString("technicianPhoneMasked").takeUnless { it.isNullOrEmpty() }
    return if (phone != null) "$name · $phone" else name
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
        Text(label, fontSize = 13.5.sp, color = Palette.muted)
        Spacer(Modifier.weight(1f))
        Text(value, fontSize = 13.5.sp, color = Palette.ink)
    }
}

@Composable
private fun TimelineCard(detail: JSONObject?, timeline: List<JSONObject>) {
    AppCard {
        CardTitle("装维进度(12 环节)", "${detail?.optInt("completedStage") ?: 0}/12 已完成 · 含合同收费")
        if (timeline.isEmpty()) {
            Text("进度加载中…", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
        }
        timeline.forEachIndexed { i, t -> TimelineItem(t, i == timeline.size - 1) }
    }
}

@Composable
private fun TimelineItem(t: JSONObject, isLast: Boolean) {
    val result = t.optString("result")
    val nodeColor = when (result) { "DONE" -> Palette.success; "DOING" -> Palette.primary; else -> Palette.line }
    Row(Modifier.height(IntrinsicSize.Min)) {
        Column(Modifier.width(14.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(Modifier.size(8.dp).background(nodeColor, CircleShape))
            if (!isLast) Box(Modifier.width(2.dp).fillMaxHeight().padding(horizontal = 3.dp).background(Palette.line))
        }
        Column(Modifier.padding(start = 8.dp, bottom = 12.dp)) {
            Text(
                "${t.optInt("stage")} ${t.optString("title")}", fontSize = 13.5.sp,
                color = if (result == "PENDING") Palette.muted else Palette.ink,
                fontWeight = if (result == "DOING") FontWeight.Bold else FontWeight.Normal,
            )
            Text(t.optString("meta"), fontSize = 12.sp, color = Palette.muted)
        }
    }
}

@Composable
private fun ActionBar(nav: Nav, no: String) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlinedButton(onClick = { nav.push(Route.Move(no)) }, modifier = Modifier.weight(1f)) { Text("变更地址", color = Palette.primary) }
        OutlinedButton(
            onClick = { scope.launch { dialTechnician(context, no) } },
            modifier = Modifier.weight(1f),
        ) { Text("联系师傅", color = Palette.primary) }
        Button(
            onClick = { scope.launch { try { OrderApi.urge(no) } catch (e: Exception) { } } },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.weight(1f),
        ) { Text("催单") }
    }
}

// 装维师傅明文电话走 GET orders/{orderNo}/technician-contact,取到后拉起拨号盘。
private suspend fun dialTechnician(context: Context, no: String) {
    try {
        val phone = OrderApi.technicianContact(no).optString("phone")
        if (phone.isBlank()) throw Api.HttpError(404, "empty phone")
        context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
    } catch (e: Exception) {
        Toast.makeText(context, "暂无法获取师傅电话,请稍后重试", Toast.LENGTH_SHORT).show()
    }
}

@Composable
private fun CancelDialog(nav: Nav, no: String, onDismiss: () -> Unit) {
    val scope = rememberCoroutineScope()
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("确认取消该订单?") },
        text = { Text("取消后将释放预占端口并关闭订单。") },
        confirmButton = {
            TextButton(onClick = {
                scope.launch {
                    try { OrderApi.cancel(no) } catch (e: Exception) { }
                    onDismiss(); nav.pop()
                }
            }) { Text("确认取消", color = Palette.err) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("再想想", color = Palette.muted) } },
    )
}
