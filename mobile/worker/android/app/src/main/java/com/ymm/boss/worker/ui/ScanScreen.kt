package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.ScanApi
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONObject

// 扫码绑定(对齐 docs/worker/scan.html):工单信息 + 模拟扫码 + 结果三态
@Composable
fun ScanScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    var result by remember { mutableStateOf<JSONObject?>(null) }
    var bindErr by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    fun doBind(epc: String, offline: Boolean) {
        scope.launch {
            result = try { ScanApi.bind(no, epc, offline) } catch (e: Exception) { bindErr = "扫码失败：${e.message}"; null }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("扫码绑定（环节 9）", onBack = { nav.pop() }, action = "在线")
        when (val s = info) {
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("加载工单失败：${s.message}", red = true) }
            is Load.Ok -> InfoCard(s.data)
            is Load.Loading -> {}
        }
        if (bindErr.isNotEmpty()) { Card(Modifier.padding(12.dp)) { Notice(bindErr, red = true) } }
        val r = result
        when (r?.optString("result")) {
            "MATCH" -> ResultOk(nav, no, r.optString("message"))
            "MISMATCH" -> ResultBad(nav, no, r.optString("message"))
            "OFFLINE_CACHED" -> ResultOffline(r.optString("message"))
            else -> ScanBox { epc, offline -> doBind(epc, offline) }
        }
        Card(Modifier.padding(12.dp)) {
            Notice("绑定结果必须与预绑定标签一致，不一致时绑定将被系统拒绝，请勿用手工数据替代扫码。")
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun InfoCard(d: JSONObject) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text(d.optString("ticketNo"), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(d.optString("statusLabel"), d.optString("status"))
        }
        KvRow("预绑定标签", d.optString("preBindTag", "-"))
        KvRow("客户", "${d.optString("customerName")} · ${d.optString("customerPhoneMasked")}")
        KvRow("安装地址", d.optString("address"))
        KvRow("分光器/端口", d.optString("splitterPort", "-"))
    }
}

@Composable
private fun ScanBox(onBind: (String, Boolean) -> Unit) {
    Card(Modifier.padding(12.dp)) {
        Text("请扫描光猫电子标签", fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink,
            modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        Spacer(Modifier.height(12.dp))
        Box(Modifier.fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(Color(0xFFE6F4FF))
            .padding(vertical = 36.dp), contentAlignment = Alignment.Center) {
            Text("▣", fontSize = 52.sp, color = Primary, fontWeight = FontWeight.Bold)
        }
        Text("将取景框对准光猫机身电子标签", fontSize = 13.sp, color = Muted,
            modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        Text("支持 LF / HF / UHF 频段 · 弱网自动离线缓存", fontSize = 12.sp, color = Muted,
            modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            SimBtn("模拟扫码", Modifier.weight(1f), primary = true) { onBind("EPC-0001", false) }
            SimBtn("模拟「不符」", Modifier.weight(1f)) { onBind("EPC-9999", false) }
        }
        SimBtn("模拟「弱网离线」", Modifier.fillMaxWidth().padding(top = 8.dp)) { onBind("EPC-0001", true) }
    }
}

@Composable
private fun ResultOk(nav: NavHost, no: String, msg: String) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text("扫码结果", fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag("一致", "DONE")
        }
        Notice(msg)
        SimBtn("继续结果上报", Modifier.fillMaxWidth().padding(top = 12.dp), primary = true) {
            nav.push(Screen.Report(no))
        }
    }
}

@Composable
private fun ResultBad(nav: NavHost, no: String, msg: String) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text("扫码结果", fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag("不一致", "DOING")
        }
        Notice(msg, red = true)
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.padding(top = 12.dp)) {
            SimBtn("换件登记", Modifier.weight(1f)) { nav.push(Screen.Replace(no)) }
            SimBtn("异常上报", Modifier.weight(1f)) { nav.push(Screen.ScanAbnormal(no)) }
        }
    }
}

@Composable
private fun ResultOffline(msg: String) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text("网络状态", fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag("弱网离线", "ACCEPTED")
        }
        Notice(msg)
    }
}

@Composable
private fun SimBtn(text: String, modifier: Modifier = Modifier, primary: Boolean = false, onClick: () -> Unit) {
    val bg = if (primary) Primary else Color.White
    val fg = if (primary) Color.White else Ink
    Text(text, fontSize = 13.sp, color = fg,
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(bg)
            .clickable { onClick() }
            .padding(vertical = 10.dp),
        textAlign = TextAlign.Center)
}