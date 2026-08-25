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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.ui.theme.Success

// 现场工具(对齐 docs/worker/tool.html):测速/光功率 + 资源查询 + 台账自查
@Composable
fun ToolScreen(nav: NavHost, no: String?) {
    val ticketNo = no ?: ""
    var refresh by remember { mutableStateOf(0) }
    val ctx = LocalContext.current
    val measure by loadOnce(ticketNo, refresh) { AssetApi.measure(ticketNo) }
    val resources by loadOnce(ticketNo, refresh) { AssetApi.resources(ticketNo) }
    val refreshLabel = stringResource(R.string.tool_action_refresh)
    val auditClean = stringResource(R.string.tool_audit_clean)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.tool_title), onBack = { nav.pop() }, action = refreshLabel, onAction = { refresh++ })
        if (ticketNo.isEmpty()) {
            Card(Modifier.padding(12.dp)) { Notice(stringResource(R.string.tool_no_ticket), red = true) }
            return@Column
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.tool_section_measure, ticketNo))
            when (val m = measure) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice(ctx.getString(R.string.tool_measure_fail, m.message ?: ""), red = true)
                is Load.Ok -> {
                    val p = m.data
                    KvRow(stringResource(R.string.tool_kv_optical), ctx.getString(R.string.tool_optical_fmt, p.optDouble("opticalPowerDbm").toString(), p.optString("opticalPowerLabel", ctx.getString(R.string.tool_normal))),
                        valueColor = Success)
                    KvRow(stringResource(R.string.tool_kv_down), ctx.getString(R.string.tool_speed_fmt, p.optDouble("downloadMbps").toString()))
                    KvRow(stringResource(R.string.tool_kv_up), ctx.getString(R.string.tool_speed_fmt, p.optDouble("uploadMbps").toString()))
                    KvRow(stringResource(R.string.tool_kv_loss), ctx.getString(R.string.tool_loss_fmt, p.optDouble("packetLossRate").toString()), valueColor = Success)
                }
            }
            Spacer(Modifier.height(8.dp))
            PrimaryButton(stringResource(R.string.tool_btn_remeasure), modifier = Modifier.fillMaxWidth()) { refresh++ }
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.tool_section_resource))
            when (val r = resources) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice(ctx.getString(R.string.tool_resource_fail, r.message ?: ""), red = true)
                is Load.Ok -> {
                    val d = r.data
                    KvRow(stringResource(R.string.tool_kv_idle_ports), ctx.getString(R.string.tool_idle_fmt, d.optInt("idlePorts")))
                    KvRow(stringResource(R.string.tool_kv_splitter), d.optString("nearestSplitter", "-"))
                    val pon = d.optJSONArray("idlePonPorts")
                    KvRow(stringResource(R.string.tool_kv_pon), pon?.let { arr -> List(arr.length()) { arr.optString(it) } }
                        ?.joinToString(" / ") ?: "-")
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.tool_section_self_audit))
            Notice(stringResource(R.string.tool_audit_notice))
            Spacer(Modifier.height(8.dp))
            PrimaryButton(stringResource(R.string.tool_btn_scan_audit), modifier = Modifier.fillMaxWidth()) { toast(ctx, auditClean) }
        }
        Spacer(Modifier.height(12.dp))
    }
}