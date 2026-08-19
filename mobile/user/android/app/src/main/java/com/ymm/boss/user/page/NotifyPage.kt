package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/notify.html:通知订阅,GET/PUT /profile/notify-settings。
private data class NotifyState(
    val promo: Boolean = true, val planRec: Boolean = false,
    val inApp: Boolean = true, val sms: Boolean = true, val push: Boolean = true,
)

private fun notifyStateFrom(s: JSONObject): NotifyState = NotifyState(
    promo = s.optJSONObject("marketing")?.optBoolean("promo", true) ?: true,
    planRec = s.optJSONObject("marketing")?.optBoolean("planRecommend", false) ?: false,
    inApp = s.optJSONObject("channels")?.optBoolean("inApp", true) ?: true,
    sms = s.optJSONObject("channels")?.optBoolean("sms", true) ?: true,
    push = s.optJSONObject("channels")?.optBoolean("push", true) ?: true,
)

@Composable
fun NotifyScreen(nav: Nav) {
    var st by remember { mutableStateOf(NotifyState()) }
    var msg by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        try { st = notifyStateFrom(ProfileApi.notifySettings()) }
        catch (e: Exception) { } // 加载失败用默认勾选,可保存覆盖
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("通知订阅设置") { nav.pop() }
        BusinessCard()
        MarketingCard(st) { st = it }
        ChannelCard(st) { st = it }
        SaveButton(msg) {
            scope.launch {
                try {
                    ProfileApi.saveNotifySettings(buildPayload(st))
                    msg = "已保存"
                } catch (e: Exception) { msg = "保存失败,请稍后重试" }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

private fun buildPayload(st: NotifyState): JSONObject =
    JSONObject()
        .put("business", JSONObject()
            .put("bill", true).put("suspendResume", true)
            .put("faultNotice", true).put("installProgress", true))
        .put("marketing", JSONObject().put("promo", st.promo).put("planRecommend", st.planRec))
        .put("channels", JSONObject().put("inApp", st.inApp).put("sms", st.sms).put("push", st.push))

@Composable
private fun BusinessCard() {
    AppCard {
        CardTitle("业务通知", "不可退订")
        CellRow("账单 / 缴费提醒", desc = "出账、到期、缴费成功", right = { Tag("开启", Palette.success) })
        CellRow("停机 / 复机提醒", desc = "欠费停机、缴费复机", right = { Tag("开启", Palette.success) })
        CellRow("故障公告", desc = "片区割接、计划维护", right = { Tag("开启", Palette.success) })
        CellRow("装维进度", desc = "订单环节变更、上门提醒", right = { Tag("开启", Palette.success) })
    }
}

@Composable
private fun MarketingCard(st: NotifyState, onChange: (NotifyState) -> Unit) {
    AppCard {
        CardTitle("营销通知", "可退订")
        ToggleCell("优惠活动推送", "每周最多 2 条", st.promo) { onChange(st.copy(promo = it)) }
        ToggleCell("套餐推荐", "升档 / 续约推荐", st.planRec) { onChange(st.copy(planRec = it)) }
    }
}

@Composable
private fun ChannelCard(st: NotifyState, onChange: (NotifyState) -> Unit) {
    AppCard {
        CardTitle("接收渠道")
        ToggleCell("站内信", null, st.inApp) { onChange(st.copy(inApp = it)) }
        ToggleCell("短信", null, st.sms) { onChange(st.copy(sms = it)) }
        ToggleCell("应用推送", null, st.push) { onChange(st.copy(push = it)) }
    }
}

@Composable
private fun ToggleCell(title: String, desc: String?, checked: Boolean, onChecked: (Boolean) -> Unit) {
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f).padding(vertical = 12.dp)) {
            Text(title, fontSize = 14.sp, color = Palette.ink)
            if (!desc.isNullOrBlank()) Text(desc, fontSize = 12.sp, color = Palette.muted)
        }
        Checkbox(checked = checked, onCheckedChange = onChecked)
    }
}

@Composable
private fun SaveButton(msg: String, onSave: () -> Unit) {
    AppCard {
        Notice(msg)
        Button(
            onClick = onSave,
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().height(44.dp),
        ) { Text("保存设置") }
    }
}
