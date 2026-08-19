package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.PlanApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

// 对应草稿 docs/user/myplan.html:当前套餐、用量摘要、变更/移机/销号入口。
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
    LaunchedEffect(Unit) {
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
        TopBar("我的套餐") { nav.pop() }
        if (err.isNotBlank()) Notice(err, Palette.err)
        CurrentPlanCard(plan)
        QuickEntries(nav, plan.planId)
        SuspendResumeCard(nav)
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
            Text(plan.name, fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink)
            Tag("在网", Palette.success)
        }
        CellRow("月费", right = { Text(plan.monthlyFee, fontSize = 13.sp, color = Palette.ink) })
        CellRow("合约到期", right = { Text(plan.contractEnd, fontSize = 13.sp, color = Palette.ink) })
        CellRow("安装地址", right = { Text(plan.installAddress, fontSize = 13.sp, color = Palette.muted) })
        CellRow("账户状态", right = { Tag("正常", Palette.success) })
        CellRow("本月账单", right = { Text(plan.currentBill, fontSize = 13.sp, color = Palette.warn) })
    }
}

@Composable
private fun QuickEntries(nav: Nav, planId: String) {
    val entries = listOf(
        "改" to Palette.primary to ("改套餐" to { nav.push(Route.Change(planId)) }),
        "增" to Palette.success to ("加购" to { nav.push(Route.Addon) }),
        "迁" to Palette.orange to ("迁址" to { nav.push(Route.Move(planId)) }),
        "退" to Palette.err to ("退订" to { nav.push(Route.Cancel(planId)) }),
    )
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 8.dp)
            .background(Palette.panel, RoundedCornerShape(12.dp)).padding(vertical = 14.dp),
    ) {
        entries.forEach { (glyphColor, labelAction) ->
            val (glyph, color) = glyphColor
            val (label, onClick) = labelAction
            Column(
                Modifier.weight(1f).clickable { onClick() },
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Box(Modifier.size(38.dp).background(color, RoundedCornerShape(10.dp)), contentAlignment = Alignment.Center) {
                    Text(glyph, color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.Bold)
                }
                Text(label, fontSize = 12.5.sp, color = Palette.ink)
            }
        }
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
