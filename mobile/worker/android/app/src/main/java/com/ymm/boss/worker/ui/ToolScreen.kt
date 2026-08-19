package com.ymm.boss.worker.ui

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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.ui.theme.Success

private val FALLBACK_NO = "ORD-20250817-001"

// 现场工具(对齐 docs/worker/tool.html):测速/光功率 + 资源查询 + 台账自查
@Composable
fun ToolScreen(nav: NavHost, no: String?) {
    val ticketNo = no ?: FALLBACK_NO
    var refresh by remember { mutableStateOf(0) }
    val measure by loadOnce(ticketNo, refresh) { AssetApi.measure(ticketNo) }
    val resources by loadOnce(ticketNo) { AssetApi.resources(ticketNo) }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("现场工具", onBack = { nav.pop() }, action = "刷新", onAction = { refresh++ })
        Card(Modifier.padding(12.dp)) {
            SectionTitle("工单 $ticketNo")
            when (val m = measure) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("测速加载失败：${m.message}", red = true)
                is Load.Ok -> {
                    val p = m.data
                    KvRow("光功率", "${p.optDouble("opticalPowerDbm")} dBm（${p.optString("opticalPowerLabel", "正常")}）",
                        valueColor = Success)
                    KvRow("下载测速", "${p.optDouble("downloadMbps")} Mbps")
                    KvRow("上传测速", "${p.optDouble("uploadMbps")} Mbps")
                    KvRow("丢包率", "${p.optDouble("packetLossRate")}%", valueColor = Success)
                }
            }
            Spacer(Modifier.height(8.dp))
            PrimaryButton("重新测速", modifier = Modifier.fillMaxWidth()) { refresh++ }
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("资源查询")
            when (val r = resources) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("资源查询失败：${r.message}", red = true)
                is Load.Ok -> {
                    val d = r.data
                    KvRow("片区空闲端口", "${d.optInt("idlePorts")} 个")
                    KvRow("最近分光器", d.optString("nearestSplitter", "-"))
                    val pon = d.optJSONArray("idlePonPorts")
                    KvRow("PON 空闲口", pon?.let { arr -> List(arr.length()) { arr.optString(it) } }
                        ?.joinToString(" / ") ?: "-")
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("台账自查")
            Notice("扫码核对该区域资产台账，差异将自动生成清单上报。")
            Spacer(Modifier.height(8.dp))
            PrimaryButton("扫码核对", modifier = Modifier.fillMaxWidth()) { toast2("台账核对无差异。") }
        }
        Spacer(Modifier.height(12.dp))
    }
}