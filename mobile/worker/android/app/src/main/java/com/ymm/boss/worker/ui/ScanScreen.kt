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
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ScanApi
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.BuildConfig
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.TagBlue
import com.ymm.boss.worker.util.CameraScanner
import com.ymm.boss.worker.util.ScanCallback
import kotlinx.coroutines.launch
import org.json.JSONObject

@Composable
fun ScanScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    var result by remember { mutableStateOf<JSONObject?>(null) }
    var bindErr by remember { mutableStateOf("") }
    var scanMode by remember { mutableStateOf(true) }
    var epcInput by remember { mutableStateOf("") }
    var cameraGranted by remember { mutableStateOf(false) }
    var cameraErr by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted -> cameraGranted = granted; if (!granted) cameraErr = ctx.getString(R.string.scan_camera_denied) }

    fun doBind(epc: String, offline: Boolean) {
        scope.launch {
            result = try { ScanApi.bind(no, epc, offline) } catch (e: Exception) { bindErr = ctx.getString(R.string.scan_err_bind, e.message ?: ""); null }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.scan_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.scan_err_load, s.message), red = true) }
            is Load.Ok -> InfoCard(s.data)
            is Load.Loading -> {}
        }
        if (bindErr.isNotEmpty()) { Card(Modifier.padding(12.dp)) { Notice(bindErr, red = true) } }
        if (cameraErr.isNotEmpty()) { Card(Modifier.padding(12.dp)) { Notice(cameraErr, red = true) } }
        val r = result
        when (r?.optString("result")) {
            "MATCH" -> ResultOk(nav, no, r.optString("message"))
            "MISMATCH" -> ResultBad(nav, no, r.optString("message"))
            "OFFLINE_CACHED" -> ResultOffline(r.optString("message"))
            else -> {
                if (scanMode) {
                    if (cameraGranted) {
                        CameraScanBox(cameraErr = { cameraErr = it }) { epc ->
                            scanMode = false; epcInput = epc; doBind(epc, false)
                        }
                    } else {
                        Card(Modifier.padding(12.dp)) {
                            Notice(stringResource(R.string.scan_camera_required))
                            PrimaryButton(stringResource(R.string.scan_camera_grant), modifier = Modifier.fillMaxWidth()) {
                                permissionLauncher.launch(android.Manifest.permission.CAMERA)
                            }
                        }
                    }
                } else {
                    ManualBindBox(epcInput, onInput = { epcInput = it }) { epc, offline -> doBind(epc, offline) }
                }
                Row(Modifier.padding(horizontal = 12.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    SimBtn(stringResource(if (scanMode) R.string.scan_btn_manual else R.string.scan_btn_camera), Modifier.weight(1f), primary = !scanMode) { scanMode = !scanMode }
                }
            }
        }
        Card(Modifier.padding(12.dp)) { Notice(stringResource(R.string.scan_bind_notice)) }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun CameraScanBox(cameraErr: (String) -> Unit, onScanned: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    Card(Modifier.padding(12.dp)) {
        Text(stringResource(R.string.scan_align_label), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink,
            modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        Spacer(Modifier.height(12.dp))
        Box(Modifier.fillMaxWidth().height(280.dp).clip(RoundedCornerShape(16.dp))) {
            CameraScanner(
                onScanned = ScanCallback { data -> scope.launch { onScanned(data.trim()) } },
                onError = { msg -> cameraErr(msg) },
            )
        }
        Spacer(Modifier.height(8.dp))
        Text(stringResource(R.string.scan_auto_submit), fontSize = 13.sp, color = Muted, modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
    }
}

@Composable
private fun ManualBindBox(epc: String, onInput: (String) -> Unit, onBind: (String, Boolean) -> Unit) {
    Card(Modifier.padding(12.dp)) {
        Text(stringResource(R.string.scan_manual_input), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink,
            modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        Spacer(Modifier.height(12.dp))
        Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(TagBlue.bg).padding(vertical = 36.dp), contentAlignment = Alignment.Center) {
            Text("▣", fontSize = 52.sp, color = Primary, fontWeight = FontWeight.Bold)
        }
        OutlinedTextField(value = epc, onValueChange = onInput, label = { Text(stringResource(R.string.scan_epc_hint)) }, singleLine = true, modifier = Modifier.fillMaxWidth())
        PrimaryButton(stringResource(R.string.scan_submit_bind), modifier = Modifier.fillMaxWidth().padding(top = 8.dp)) {
            onBind(epc.trim(), false)
        }
        Text(stringResource(R.string.scan_freq_hint), fontSize = 12.sp, color = Muted, modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        Spacer(Modifier.height(12.dp))
        if (BuildConfig.DEBUG) {
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                SimBtn(stringResource(R.string.scan_sim_match), Modifier.weight(1f), primary = true) { onBind("EPC-0001", false) }
                SimBtn(stringResource(R.string.scan_sim_mismatch), Modifier.weight(1f)) { onBind("EPC-9999", false) }
            }
            SimBtn(stringResource(R.string.scan_sim_offline), Modifier.fillMaxWidth().padding(top = 8.dp)) { onBind("EPC-0001", true) }
        }
    }
}

@Composable
private fun InfoCard(d: JSONObject) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(d.optString("ticketNo"), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(d.optString("statusLabel"), d.optString("status"))
        }
        KvRow(stringResource(R.string.scan_prebind), d.optString("preBindTag", "-"))
        KvRow(stringResource(R.string.td_customer), "${d.optString("customerName")} · ${d.optString("customerPhoneMasked")}")
        KvRow(stringResource(R.string.scan_kv_address), d.optString("address")); KvRow(stringResource(R.string.scan_kv_splitter), d.optString("splitterPort", "-"))
    }
}

@Composable
private fun ResultOk(nav: NavHost, no: String, msg: String) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.scan_result_title), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(stringResource(R.string.scan_result_match), "DONE")
        }
        Notice(msg)
        SimBtn(stringResource(R.string.scan_continue_report), Modifier.fillMaxWidth().padding(top = 12.dp), primary = true) { nav.push(Screen.Report(no)) }
    }
}

@Composable
private fun ResultBad(nav: NavHost, no: String, msg: String) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.scan_result_title), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(stringResource(R.string.scan_result_mismatch), "DOING")
        }
        Notice(msg, red = true)
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.padding(top = 12.dp)) {
            SimBtn(stringResource(R.string.scan_replace), Modifier.weight(1f)) { nav.push(Screen.Replace(no)) }
            SimBtn(stringResource(R.string.scan_abnormal), Modifier.weight(1f)) { nav.push(Screen.ScanAbnormal(no)) }
        }
    }
}

@Composable
private fun ResultOffline(msg: String) {
    Card(Modifier.padding(12.dp)) {
        Row(Modifier.fillMaxWidth().padding(bottom = 8.dp), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.scan_offline_title), fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
            StatusTag(stringResource(R.string.scan_offline_label), "ACCEPTED")
        }
        Notice(msg)
    }
}

@Composable
private fun SimBtn(text: String, modifier: Modifier = Modifier, primary: Boolean = false, onClick: () -> Unit) {
    val bg = if (primary) Primary else Color.White
    val fg = if (primary) Color.White else Ink
    Text(text, fontSize = 13.sp, color = fg, modifier = modifier.clip(RoundedCornerShape(8.dp)).background(bg).clickable { onClick() }.padding(vertical = 10.dp), textAlign = TextAlign.Center)
}
