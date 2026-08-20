package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.FaultApi
import com.ymm.boss.user.api.toObjList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/faultdetail.html:报修进度(no) + 报障 6 环节时间轴。
@Composable
fun FaultDetailScreen(nav: Nav, no: String) {
    var detail by remember { mutableStateOf<JSONObject?>(null) }
    var loadErr by remember { mutableStateOf("") }
    var reloadKey by remember { mutableStateOf(0) }

    LaunchedEffect(no, reloadKey, nav.refreshTick) {
        try {
            detail = FaultApi.detail(no)
        } catch (e: Exception) { loadErr = "报修单加载失败" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("报修详情", onBack = { nav.pop() }, action = "评价", onAction = { nav.push(Route.Rate(no)) })
        val d = detail
        if (d == null) {
            AppCard { CardTitle(if (loadErr.isBlank()) "加载中…" else loadErr) }
        } else {
            val fault = d.optJSONObject("fault") ?: JSONObject()
            InfoCard(fault, d)
            TimelineCard(d.optJSONArray("timeline").toObjList())
            ActionsCard(no) { reloadKey++ }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun InfoCard(fault: JSONObject, d: JSONObject) {
    AppCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            CardTitle(fault.optString("ticketNo", "—"))
            Spacer(Modifier.width(8.dp))
            Tag(fault.optString("statusLabel", "—"), Palette.primary)
        }
        CellRow("故障类型", right = { InfoVal(fault.optString("faultTypeLabel", "—")) })
        CellRow("故障地址", right = { InfoVal(fault.optString("address", "—")) })
        CellRow("报修时间", right = { InfoVal(fault.optString("createdAt", "—")) })
        val tech = buildString {
            append(d.optString("technicianName").ifBlank { "—" })
            d.optString("technicianPhoneMasked").takeIf { it.isNotBlank() }?.let { append(" · ").append(it) }
        }
        CellRow("受理师傅", right = { InfoVal(tech) })
        CellRow("SLA", right = { InfoVal(d.optString("sla").ifBlank { "—" }) })
    }
}

@Composable
private fun InfoVal(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.ink, modifier = Modifier.padding(start = 12.dp))
}

@Composable
private fun TimelineCard(timeline: List<JSONObject>) {
    AppCard {
        CardTitle("处理进度（报障 6 环节）")
        if (timeline.isEmpty()) {
            EmptyState("暂无进度")
        }
        timeline.forEachIndexed { i, t ->
            val heading = t.optInt("step").let { if (it > 0) "$it " else "" } + t.optString("title")
            TimelineItem(heading, t.optString("result", "PENDING"), t.optString("meta"), isLast = i == timeline.size - 1)
        }
    }
}

@Composable
private fun TimelineItem(title: String, result: String, meta: String, isLast: Boolean) {
    Row(Modifier.fillMaxWidth().padding(top = 12.dp)) {
        Column(Modifier.width(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            val color = when (result) {
                "DONE" -> Palette.success
                "DOING" -> Palette.primary
                else -> Palette.subtle
            }
            Box(
                Modifier.size(10.dp).background(color, CircleShape),
            )
            if (!isLast) {
                Column(
                    Modifier.width(2.dp).height(32.dp)
                        .background(if (result == "DONE") Palette.success else Palette.line),
                ) {}
            }
        }
        Column(Modifier.padding(start = 12.dp)) {
            Text(title, fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink)
            if (meta.isNotBlank()) Text(meta, fontSize = 12.sp, color = Palette.muted)
        }
    }
}

// 联系师傅:GET contact 取明文电话后拉起拨号盘(404 回落 toast);催单:POST urge 后重载详情。
@Composable
private fun ActionsCard(no: String, onReload: () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 6.dp)) {
        OutlinedButton(
            onClick = { scope.launch { contactTechnician(context, no) } },
            modifier = Modifier.weight(1f).height(42.dp),
        ) { Text("联系师傅", color = Palette.primary) }
        Spacer(Modifier.width(12.dp))
        Button(
            onClick = { scope.launch { urge(context, no, onReload) } },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.weight(1f).height(42.dp),
        ) { Text("催单", color = Color.White) }
    }
}

private suspend fun contactTechnician(context: Context, no: String) {
    try {
        val phone = FaultApi.contact(no).optString("technicianPhone")
        if (phone.isBlank()) throw Api.HttpError(404, "empty phone")
        context.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
    } catch (e: Exception) {
        toast(context, "暂未指派师傅,无法获取联系电话")
    }
}

private suspend fun urge(context: Context, no: String, onReload: () -> Unit) {
    try {
        FaultApi.urge(no)
        toast(context, "已提交催单,请耐心等待")
    } catch (e: Exception) {
        toast(context, "催单失败,请稍后重试")
    }
    onReload()
}

private fun toast(context: Context, text: String) {
    Toast.makeText(context, text, Toast.LENGTH_SHORT).show()
}
