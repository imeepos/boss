package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar

/** 对应草稿 docs/user/help.html:帮助中心(静态热门问题+业务办理)。 */
@Composable
fun HelpScreen(nav: Nav) {
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("帮助中心", onBack = { nav.pop() }, action = "在线客服", onAction = { nav.push(Route.Service) })
        AppCard {
            CardTitle("热门问题")
            HotEntries(nav)
        }
        AppCard {
            CardTitle("业务办理")
            BizEntries(nav)
        }
        AppCard {
            Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally) {
                Text("客服热线 10086 · 服务时间 8:00~22:00", fontSize = 12.5.sp, color = Palette.muted)
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun HotEntries(nav: Nav) {
    val entries = listOf(
        Triple("账单怎么看？", "账单明细与费用构成") { nav.push(Route.Bills) },
        Triple("如何修改套餐？", "升档/降档/预约生效") { nav.push(Route.Change("")) },
        Triple("如何开发票？", "电子发票开票流程") { nav.push(Route.Invoice) },
        Triple("如何迁址移机？", "迁址资源核查与工单") { nav.push(Route.Move("")) },
        Triple("上网故障如何自检？", "自助排障引导") { nav.push(Route.Diy) },
    )
    entries.forEach { (t, d, onClick) -> CellRow(title = t, desc = d, onClick = onClick, right = { Text("›", color = Palette.subtle) }) }
}

@Composable
private fun BizEntries(nav: Nav) {
    val entries = listOf(
        "如何充值？" to { nav.push(Route.Topup) },
        "如何退订拆机？" to { nav.push(Route.Cancel("")) },
        "如何更换绑定手机号？" to { nav.push(Route.Security) },
    )
    entries.forEach { (t, onClick) -> CellRow(title = t, onClick = onClick, right = { Text("›", color = Palette.subtle) }) }
}
