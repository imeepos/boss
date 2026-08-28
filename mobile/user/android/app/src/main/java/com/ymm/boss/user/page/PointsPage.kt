package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.PointsApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.TabHeader
import com.ymm.boss.user.ui.rememberSubmitGuard
import kotlinx.coroutines.launch
import org.json.JSONObject

// 我的积分页,契约 api/openapi/user/loy.yaml:余额/等级概览 + 赚积分任务 + 积分明细 + 兑换。
// 卡片组件拆在 PointsCards.kt(本文件仅保留 Screen 组装,≤300 行红线)。
// 任务完成周期内幂等;兑换先扣后发,成功余额刷新 + 券到账终态文案。

@Composable
fun PointsScreen(nav: Nav) {
    var overview by remember { mutableStateOf<JSONObject?>(null) }
    var tierName by remember { mutableStateOf("") }
    var tasks by remember { mutableStateOf(emptyList<JSONObject>()) }
    var offers by remember { mutableStateOf(emptyList<JSONObject>()) }
    var loadErr by remember { mutableStateOf("") }
    var notice by remember { mutableStateOf("") }
    var exchangeErr by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val guard = rememberSubmitGuard()
    var confirmOffer by remember { mutableStateOf<JSONObject?>(null) }

    suspend fun load() {
        loadErr = ""
        try { overview = PointsApi.overview() } catch (e: Exception) { loadErr = "积分加载失败，请检查网络" }
        try { tierName = PointsApi.tier().optJSONObject("tier")?.optString("name").orEmpty() } catch (e: Exception) {}
        try { tasks = PointsApi.tasks().toObjectList() } catch (e: Exception) {}
        try { offers = PointsApi.exchangeOffers().toObjectList() } catch (e: Exception) {}
    }
    LaunchedEffect(nav.refreshTick) { load() }

    fun doExchange(templateId: Long) {
        if (!guard.acquire()) return
        exchangeErr = ""
        notice = ""
        scope.launch {
            try {
                PointsApi.exchange(templateId)
                notice = "兑换成功，优惠券已放入券仓"
                load() // 刷新余额/流水
            } catch (e: Exception) {
                exchangeErr = Api.friendlyMessage(e).ifBlank { "兑换失败，请稍后重试" }
            } finally { guard.release() }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        // 一级 tab 页(Nav.TABS 第 5 项):标题栏用 TabHeader,无返回键,对齐订单 tab 标准。
        TabHeader("我的积分")
        if (loadErr.isNotEmpty()) {
            AppCard {
                Text(loadErr, fontSize = 13.sp, color = Palette.err)
                TextButton(onClick = { scope.launch { load() } }) { Text("重试") }
            }
        }
        BalanceCard(overview, tierName)
        ExchangeCard(
            offers = offers,
            balance = overview?.optLong("balance", 0) ?: 0L,
            notice = notice,
            err = exchangeErr,
            onExchange = { o -> confirmOffer = offers.firstOrNull { it.optLong("templateId") == o } },
        )
        TaskCard(tasks, onDone = { scope.launch { load() } })
        EntriesCard(overview)
        Spacer(Modifier.height(12.dp))
    }
    confirmOffer?.let { o ->
        androidx.compose.material3.AlertDialog(
            onDismissRequest = { confirmOffer = null },
            title = { Text("确认兑换") },
            text = { Text("将消耗 ${o.optLong("pointsPrice")} 积分兑换「${o.optString("name")}」？") },
            confirmButton = {
                TextButton(onClick = {
                    confirmOffer = null
                    doExchange(o.optLong("templateId"))
                }) { Text("确认兑换", color = Palette.primary) }
            },
            dismissButton = { TextButton(onClick = { confirmOffer = null }) { Text("取消") } },
        )
    }
}