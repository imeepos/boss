package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/** 对应草稿 docs/user/coupon.html:优惠券与活动。GET /coupons。 */
@Composable
fun CouponScreen(nav: Nav) {
    var status by remember { mutableStateOf("available") }
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var inviteLink by remember { mutableStateOf("") }
    var msg by remember { mutableStateOf("") }

    LaunchedEffect(status, nav.refreshTick) {
        try {
            val d = BillApi.coupons(status)
            items = d.optJSONArray("items").optList()
            inviteLink = d.optString("inviteLink")
        } catch (e: Exception) { /* 骨架保留空列表 */ }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("优惠券与活动", onBack = { nav.pop() })

        Tabs(status) { status = it }
        if (items.isEmpty()) {
            EmptyState("暂无优惠券")
        }
        items.forEach { c -> CouponCard(c, status) { nav.push(Route.Products) } }

        AppCard {
            Text("邀请好友", fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink)
            Text("邀请好友办理宽带,双方各得 ¥50 优惠券。", fontSize = 12.5.sp, color = Palette.muted,
                modifier = Modifier.padding(vertical = 8.dp))
            if (msg.isNotEmpty()) Text(msg, fontSize = 12.sp, color = Palette.primary,
                modifier = Modifier.padding(bottom = 6.dp))
            OutlinedButton(onClick = { msg = "邀请链接:" + inviteLink.ifBlank { "生成中" } }) {
                Text("复制邀请链接", color = Palette.primary)
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun Tabs(current: String, onSelect: (String) -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 8.dp)) {
        listOf("available" to "可用", "used" to "已使用", "expired" to "已过期").forEach { (key, label) ->
            val active = key == current
            Text(
                label,
                fontSize = 13.5.sp,
                fontWeight = if (active) FontWeight.Bold else FontWeight.Normal,
                color = if (active) Palette.primary else Palette.muted,
                modifier = Modifier.padding(end = 18.dp).clickable { onSelect(key) },
            )
        }
    }
}

@Composable
private fun CouponCard(c: JSONObject, status: String, onUse: () -> Unit) {
    val threshold = c.optDouble("threshold", 0.0)
    AppCard(Modifier.fillMaxWidth()) {
        Row {
            Box(Modifier.width(4.dp).height(72.dp).background(Palette.primary, RoundedCornerShape(2.dp)))
            Column(Modifier.padding(start = 12.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("¥" + trimZero(c.optDouble("amount")), fontSize = 18.sp,
                        fontWeight = FontWeight.Bold, color = Palette.ink)
                    if (threshold > 0) {
                        Text("  满 " + trimZero(threshold) + " 可用", fontSize = 12.sp, color = Palette.muted)
                    }
                    Spacer(Modifier.weight(1f))
                    Tag(statusLabel(status), if (status == "available") Palette.primary else Palette.muted)
                }
                Text(
                    c.optString("title") + " · 有效期至 " + c.optString("expireAt"),
                    fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(top = 6.dp),
                )
                if (status == "available") {
                    Button(
                        onClick = onUse,
                        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                        modifier = Modifier.padding(top = 10.dp),
                    ) { Text("去使用", fontSize = 12.5.sp) }
                }
            }
        }
    }
}

private fun statusLabel(status: String): String = when (status) {
    "available" -> "可用"; "used" -> "已使用"; "expired" -> "已过期"; else -> status
}

private fun trimZero(v: Double): String =
    if (v == v.toLong().toDouble()) v.toLong().toString() else "%.2f".format(v)

private fun org.json.JSONArray?.optList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
