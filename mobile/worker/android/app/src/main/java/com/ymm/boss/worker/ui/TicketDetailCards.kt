package com.ymm.boss.worker.ui

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.res.stringResource
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.ui.theme.Err
import com.ymm.boss.worker.ui.theme.ErrBorder
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Panel
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.SuccessBorder
import com.ymm.boss.worker.ui.theme.Warn
import org.json.JSONArray
import org.json.JSONObject

/**
 * 工单详情卡片集(对齐 designs/worker-order-detail-v1.spec.md):
 * - 工单头(按状态/类型分支)
 * - 快捷动作(3 列等分)
 * - 装维进度时间轴(横版,12/6 节点)
 * - 四码校验 2×2
 * - 风控校验
 * - 完成回执(D 屏专属)
 * - 底栏动作区(sticky)
 *
 * 类型(INSTALL/REPAIR)由 stages.length 推断(后端 TicketDetail 暂未返 type)。
 */

// 非组合上下文取资源
private fun s(res: Int, vararg fmt: Any = emptyArray()) = Api.context().getString(res, *fmt)

// Card1 工单头(状态/类型分支)
@Composable
internal fun DetailHeaderCard(d: JSONObject, type: String) {
    val status = d.optString("status")
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically) {
            Text(d.optString("ticketNo"), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(d.optString("statusLabel"), status)
        }
        if (type == "REPAIR") {
            val fault = d.optString("faultTypeLabel")
            if (fault.isNotEmpty()) Text(fault, fontSize = 15.sp, fontWeight = FontWeight.Bold,
                color = Err, modifier = Modifier.padding(bottom = 4.dp))
        } else {
            val product = d.optString("product")
            if (product.isNotEmpty()) KvRow(stringResource(R.string.td_product), product)
        }
        val name = d.optString("customerName")
        val phone = d.optString("customerPhoneMasked")
        if (name.isNotEmpty() || phone.isNotEmpty()) {
            KvRow(stringResource(R.string.td_customer), listOf(name, phone).filter { it.isNotEmpty() }.joinToString(" "))
        }
        val addr = d.optString("address")
        if (addr.isNotEmpty()) KvRow(stringResource(R.string.td_install_addr), addr)
        if (type != "REPAIR") {
            val sp = d.optString("splitterPort")
            if (sp.isNotEmpty()) KvRow(stringResource(R.string.td_splitter), sp)
            val pb = d.optString("preBindTag")
            if (pb.isNotEmpty()) KvRow(stringResource(R.string.td_prebind), pb)
            val ss = d.optString("scheduleSlot")
            if (ss.isNotEmpty()) KvRow(stringResource(R.string.td_schedule), ss)
            val dist = d.optDouble("distanceKm", -1.0)
            if (status == "TODO" && dist >= 0) KvRow(stringResource(R.string.td_distance), "%.1f km".format(dist), valueColor = Primary)
        } else {
            val reported = d.optString("reportedAt")
            if (reported.isNotEmpty()) KvRow(stringResource(R.string.td_reported_at), reported)
            val sla = d.optInt("slaLeftMinutes", -1)
            if (sla >= 0) {
                KvRow(stringResource(R.string.td_sla_left), "%d:%02d".format(sla / 60, sla % 60),
                    valueColor = Warn)
            }
            val diag = d.optString("remoteDiagnosis")
            if (diag.isNotEmpty()) {
                Text(stringResource(R.string.td_remote_diag, diag), fontSize = 12.sp, color = Muted,
                    modifier = Modifier.padding(top = 4.dp))
            }
        }
        if (status == "DONE") {
            val finished = d.optString("finishedAt")
            if (finished.isNotEmpty()) KvRow(stringResource(R.string.td_finished_at), finished)
        }
    }
}

// Card2 快捷动作(3 列等分;TODO 态隐藏,只需"领取工单"主按钮)
@Composable
internal fun QuickActionRow(d: JSONObject, nav: NavHost, no: String) {
    val ctx = LocalContext.current
    val phone = d.optString("customerPhoneMasked")
    Card(Modifier.padding(12.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            ActionBtn(stringResource(R.string.td_contact_customer), Modifier.weight(1f)) {
                if (phone.isBlank()) toast(ctx, s(R.string.td_phone_masked))
                else ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
            }
            ActionBtn(stringResource(R.string.td_navi), Modifier.weight(1f)) { nav.push(Screen.Navi(no)) }
            ActionBtn(stringResource(R.string.td_checkin), Modifier.weight(1f)) { nav.push(Screen.Checkin(no)) }
        }
    }
}

// Card3 装维进度时间轴(横版一字排开)
// total 用 spec §4.4 锁定的环节总数(12/6),不取 stages.length();
// 后端 TicketDetail 暂未返 stageTotal,前端由 inferTicketType 推断。
// 文案对齐 spec §4.4:"装维进度(当前:N 环节 · M/N 已完成)"
@Composable
internal fun TimelineCard(d: JSONObject) {
    val stages = d.optJSONArray("stages") ?: JSONArray()
    val total = if (inferTicketType(d) == "REPAIR") 6 else 12
    val doneCount = (0 until stages.length())
        .count { stages.optJSONObject(it).optString("result") == "DONE" }
    val currentStage = stages.optJSONObject(stages.length() - 1)?.optInt("stage") ?: 0
    Card(Modifier.padding(12.dp)) {
        SectionTitle(stringResource(R.string.td_progress),
            more = stringResource(R.string.td_progress_more, currentStage, doneCount, total))
        Row(modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically) {
            for (i in 1..total) {
                // 按环节号 1..N 在 stages 里找匹配日志,缺环节视为 PENDING
                val stage = (0 until stages.length())
                    .map { stages.optJSONObject(it) }
                    .firstOrNull { it.optInt("stage") == i }
                val st = if (stage == null) NodeState.PENDING
                         else nodeStateOf(stage.optString("result"))
                NodeDot(st, i)
                if (i < total) {
                    val nextStage = (0 until stages.length())
                        .map { stages.optJSONObject(it) }
                        .firstOrNull { it.optInt("stage") == i + 1 }
                    val nextSt = if (nextStage == null) NodeState.PENDING
                                 else nodeStateOf(nextStage.optString("result"))
                    TimelineConnector(st == NodeState.DONE && nextSt == NodeState.DONE,
                        Modifier.weight(1f))
                }
            }
        }
    }
}

// Card4 四码校验(2×2):按 spec §4.5 status 三态分色
//   LINKED   → 绿 + "已通过校验"
//   CONFLICT → 红 + "不一致 · 需核实"
//   UNLINKED → 灰 + "待校验"
@Composable
internal fun QuadCard(quad: JSONObject?) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle(stringResource(R.string.td_quad_title), more = stringResource(R.string.td_quad_more))
        if (quad == null) { Notice(stringResource(R.string.td_quad_missing)); return@Card }
        val status = quad.optString("status")
        val (cellColor, stText) = when (status) {
            "CONFLICT" -> Err to stringResource(R.string.td_quad_conflict)
            "LINKED"   -> Success to stringResource(R.string.td_quad_passed)
            else       -> Muted to stringResource(R.string.td_quad_pending)
        }
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                QuadCell(stringResource(R.string.td_asset), quad.optString("assetCode"), stText, cellColor, Modifier.weight(1f))
                QuadCell(stringResource(R.string.td_user_code), quad.optString("customerCode"), stText, cellColor, Modifier.weight(1f))
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                QuadCell(stringResource(R.string.td_port_code), quad.optString("portCode"), stText, cellColor, Modifier.weight(1f))
                QuadCell(stringResource(R.string.td_addr_code), quad.optString("addrCode"), stText, cellColor, Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun QuadCell(label: String, value: String, st: String, color: Color, modifier: Modifier) {
    val border = when (color) {
        Success -> SuccessBorder
        Err -> ErrBorder
        else -> Line
    }
    Box(modifier = modifier.background(Panel, RoundedCornerShape(8.dp))
        .border(1.dp, border, RoundedCornerShape(8.dp)).padding(12.dp)) {
        Column {
            Text(label, fontSize = 12.sp, color = Muted)
            Text(value.ifEmpty { "-" }, fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Ink)
            Text(st, fontSize = 12.sp, color = color)
        }
    }
}

// Card5 风控校验
@Composable
internal fun RiskCard(risk: JSONObject?) {
    Card(Modifier.padding(12.dp)) {
        SectionTitle(stringResource(R.string.td_risk_title), more = stringResource(R.string.td_risk_more))
        val hit = risk?.optBoolean("blacklistHit", false) == true
                || risk?.optBoolean("graylistHit", false) == true
        val color = if (hit) Err else Success
        val text = if (hit) stringResource(R.string.td_risk_hit) else stringResource(R.string.td_risk_miss)
        Row(Modifier.fillMaxWidth().padding(vertical = 6.dp),
            horizontalArrangement = Arrangement.SpaceBetween) {
            Text(if (hit) stringResource(R.string.td_risk_hit_short) else stringResource(R.string.td_risk_miss_short), fontSize = 13.sp, color = color,
                fontWeight = FontWeight.Medium)
            Text(text, fontSize = 13.sp, color = color)
        }
        Notice(stringResource(R.string.td_risk_notice))
    }
}

// Card6 完成回执(D 屏专属;签名位按 spec §6 不放预置签名)
@Composable
internal fun ReceiptCard() {
    Card(Modifier.padding(12.dp)) {
        SectionTitle(stringResource(R.string.td_receipt_title))
        Row(verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.padding(vertical = 4.dp)) {
            Box(modifier = Modifier.background(Success, androidx.compose.foundation.shape.CircleShape)
                .padding(4.dp)) {
                Text("✓", color = Panel, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
            Text(stringResource(R.string.td_receipt_confirmed), fontSize = 14.sp, fontWeight = FontWeight.SemiBold,
                color = Success)
        }
        Spacer(Modifier.height(8.dp))
        Text(stringResource(R.string.td_signature), fontSize = 12.sp, color = Muted)
        Box(modifier = Modifier.fillMaxWidth().height(56.dp)
            .border(1.dp, Line, RoundedCornerShape(8.dp))
            .padding(8.dp), contentAlignment = Alignment.Center) {
            Text(stringResource(R.string.td_signature_placeholder), fontSize = 12.sp, color = Muted)
        }
    }
}

// 底栏动作区:按状态分支(sticky)
@Composable
internal fun BottomActionBar(d: JSONObject, nav: NavHost, no: String,
                              onAccept: () -> Unit, onRollback: () -> Unit, onRetry: () -> Unit) {
    val status = d.optString("status")
    val type = inferTicketType(d)
    Row(modifier = Modifier.fillMaxWidth().background(Panel)
        .padding(horizontal = 12.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        when {
            status == "TODO" -> PrimaryAction(stringResource(R.string.orders_take_btn), Modifier.weight(1f), onClick = onAccept)
            status == "DONE" -> {
                OutlinedAction(stringResource(R.string.td_view_report), Modifier.weight(1f)) { nav.push(Screen.Report(no)) }
                OutlinedAction(stringResource(R.string.td_back_list), Modifier.weight(1f)) { nav.pop() }
            }
            type == "REPAIR" -> {
                OutlinedAction(stringResource(R.string.td_contact_customer), Modifier.weight(1f)) { nav.push(Screen.Service) }
                OutlinedAction(stringResource(R.string.td_reschedule), Modifier.weight(1f)) { nav.push(Screen.Reschedule(no)) }
                PrimaryAction(stringResource(R.string.td_repair_report), Modifier.weight(1f)) { nav.push(Screen.RepairReport(no)) }
            }
            else -> {
                OutlinedAction(stringResource(R.string.td_abnormal), Modifier.weight(1f)) { nav.push(Screen.ScanAbnormal(no)) }
                OutlinedAction(stringResource(R.string.td_transfer), Modifier.weight(1f)) { nav.push(Screen.Transfer(no)) }
                OutlinedAction(stringResource(R.string.td_reschedule), Modifier.weight(1f)) { nav.push(Screen.Reschedule(no)) }
                PrimaryAction(stringResource(R.string.td_scan_bind), Modifier.weight(1f)) { nav.push(Screen.Scan(no)) }
            }
        }
    }
}

// A 屏时间轴卡内次按钮(回退/重试,spec §4.7)
@Composable
internal fun TimelineExtras(no: String, onRollback: () -> Unit, onRetry: () -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp)) {
        OutlinedAction(stringResource(R.string.td_rollback), Modifier.weight(1f), onClick = onRollback)
        OutlinedAction(stringResource(R.string.td_retry_stage), Modifier.weight(1f), onClick = onRetry)
    }
}