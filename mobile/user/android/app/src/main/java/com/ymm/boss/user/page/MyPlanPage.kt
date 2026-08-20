package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.AddCircle
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.PlanApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.IconTile
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 对应草稿 docs/user/myplan.html:当前套餐、用量摘要、变更/移机/销号入口。
// 视觉基准与四个 tab 页对齐:快捷入口用 IconTile(浅底同色图标),金额 Bold。
private data class PlanInfo(
    val planId: String = "",
    val name: String = "加载中…",
    val monthlyFee: String = "—",
    val contractEnd: String = "—",
    val installAddress: String = "—",
    val currentBill: String = "—",
)

@Composable
fun MyPlanScreen(nav: Nav) {
    var plan by remember { mutableStateOf(PlanInfo()) }
    var err by remember { mutableStateOf("") }
    LaunchedEffect(nav.refreshTick) {
        try {
            val p = PlanApi.profile().optJSONObject("plan") ?: JSONObject()
            plan = PlanInfo(
                planId = p.optString("planId"),
                name = p.optString("name", "—"),
                monthlyFee = "¥${p.optString("monthlyFee")}/月",
                contractEnd = p.optString("contractEnd").ifBlank { "—" },
                installAddress = p.optString("installAddress").ifBlank { "—" },
                currentBill = currentBillText(p),
            )
        } catch (e: Exception) { err = "套餐信息加载失败,请稍后重试" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("我的套餐", onBack = { nav.pop() })
        if (err.isNotBlank()) Notice(err, Palette.err)
        CurrentPlanCard(plan)
        QuickEntries(nav, plan.planId)
        SuspendResumeCard(nav)
        Spacer(Modifier.height(12.dp))
    }
}

// 本月账单:plan.currentBillAmount/currentBillDue,任一缺失显示 "—"。
private fun currentBillText(p: JSONObject): String {
    if (p.isNull("currentBillAmount")) return "—"
    val amount = "¥" + "%.2f".format(p.optDouble("currentBillAmount"))
    val due = p.optString("currentBillDue").takeUnless { it.isBlank() || p.isNull("currentBillDue") }
    return if (due != null) "$amount · $due" else amount
}

@Composable
private fun CurrentPlanCard(plan: PlanInfo) {
    AppCard(Modifier.border(1.dp, Palette.primary, RoundedCornerShape(12.dp))) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(plan.name, fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink, modifier = Modifier.weight(1f))
            Tag("在网", Palette.success)
        }
        CellRow("月费", right = { AmountText(plan.monthlyFee) })
        CellRow("合约到期", right = { Text(plan.contractEnd, fontSize = 13.sp, color = Palette.ink) })
        CellRow("安装地址", right = { Text(plan.installAddress, fontSize = 13.sp, color = Palette.muted) })
        CellRow("账户状态", right = { Tag("正常", Palette.success) })
        CellRow("本月账单", right = { AmountText(plan.currentBill, color = Palette.warn) })
    }
}

/** 金额一律 Bold,与基准页价格形态一致。 */
@Composable
private fun AmountText(text: String, color: Color = Palette.ink) {
    Text(text, fontSize = 13.sp, fontWeight = FontWeight.Bold, color = color)
}

@Composable
private fun QuickEntries(nav: Nav, planId: String) {
    val entries: List<Pair<Pair<ImageVector, Color>, Pair<String, () -> Unit>>> = listOf(
        (Icons.Filled.SwapHoriz to Palette.primary) to ("改套餐" to { nav.push(Route.Change(planId)) }),
        (Icons.Filled.AddCircle to Palette.success) to ("加购" to { nav.push(Route.Addon) }),
        (Icons.AutoMirrored.Filled.Send to Palette.orange) to ("迁址" to { nav.push(Route.Move(planId)) }),
        (Icons.Filled.Cancel to Palette.err) to ("退订" to { nav.push(Route.Cancel(planId)) }),
    )
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 6.dp)
            .background(Palette.panel, RoundedCornerShape(12.dp)).padding(vertical = 12.dp),
    ) {
        entries.forEach { (iconColor, labelAction) ->
            val (icon, tint) = iconColor
            val (label, onClick) = labelAction
            QuickEntry(icon, tint, label, Modifier.weight(1f), onClick = onClick)
        }
    }
}

/** 快捷入口形态与 ProfilePage.QuickEntry 一致:IconTile 40dp/12dp + 13sp W500 标签。 */
@Composable
private fun QuickEntry(
    icon: ImageVector, tint: Color, label: String,
    modifier: Modifier = Modifier, onClick: () -> Unit,
) {
    Column(modifier.clickable { onClick() }, horizontalAlignment = Alignment.CenterHorizontally) {
        IconTile(icon, tint, size = 40.dp, corner = 12.dp)
        Spacer(Modifier.height(8.dp))
        Text(label, fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink)
    }
}

@Composable
private fun SuspendResumeCard(nav: Nav) {
    AppCard {
        CardTitle("停机 / 复机状态")
        CellRow("当前状态", "无欠费,网络正常", right = { Tag("正常", Palette.success) })
        CellRow("缴费复机", "欠费停机后缴费自动复机,3 分钟内恢复", onClick = { nav.push(Route.Pay) }, right = {
            Text("去缴费", fontSize = 13.sp, color = Palette.primary)
        })
    }
}
