package com.ymm.boss.worker.ui

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.TicketApi

// 一键导航(对齐 docs/worker/navi.html)
@Composable
fun NaviScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.navi(no) }
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("一键导航", onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("导航信息加载失败，请刷新重试。", red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow("工单号", d.optString("ticketNo"))
                    KvRow("地址", d.optString("address"))
                    if (!d.isNull("distanceKm")) KvRow("距离", "${d.optDouble("distanceKm")} km")
                    if (!d.isNull("lat") && !d.isNull("lng")) {
                        KvRow("坐标", "${d.optDouble("lat")}, ${d.optDouble("lng")}")
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    PrimaryButton("打开外部地图", modifier = Modifier.fillMaxWidth()) {
                        val lat = d.optDouble("lat")
                        val lng = d.optDouble("lng")
                        val uri = if (lat != 0.0 && lng != 0.0) "geo:$lat,$lng?q=$lat,$lng"
                        else "geo:0,0?q=${Uri.encode(d.optString("address"))}"
                        ctx.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(uri)))
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}