package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.SurveyApi
import com.ymm.boss.worker.util.LocationHelper
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.launch
import org.json.JSONArray
import java.util.UUID

// 勘测任务详情(W7):接单/现场回填(打点坐标+照片+设施状态备注+建议),append-only。
@Composable
fun SurveyDetailScreen(nav: NavHost, id: Long) {
    var refresh by remember { mutableStateOf(0) }
    val state by loadOnce(id, refresh) { SurveyApi.detail(id) }
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var note by remember { mutableStateOf("") }
    var suggestion by remember { mutableStateOf("CAN_INSTALL") }
    var lat by remember { mutableStateOf(0.0) }
    var lng by remember { mutableStateOf(0.0) }
    var locating by remember { mutableStateOf(false) }
    var photoIds by remember { mutableStateOf(listOf<Long>()) }
    var uploading by remember { mutableStateOf(false) }
    var submitting by remember { mutableStateOf(false) }

    // 打点:进页自动取一次定位,可手动重取。
    fun locate() {
        scope.launch {
            locating = true
            val r = LocationHelper.getCurrentLocation(ctx)
            if (r != null) { lat = r.lat; lng = r.lng } else toast(ctx, ctx.getString(R.string.sv_locate_fail))
            locating = false
        }
    }
    LaunchedEffect(id) { locate() }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        uploading = true
        scope.launch {
            try {
                val bytes = ctx.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                if (bytes == null) throw IllegalStateException("read fail")
                val r = SurveyApi.uploadPhoto("survey.jpg", ctx.contentResolver.getType(uri) ?: "image/jpeg", bytes)
                val attId = r.optLong("id")
                if (attId > 0) photoIds = photoIds + attId
                toast(ctx, ctx.getString(R.string.photo_toast_ok))
            } catch (e: Exception) { toast(ctx, e.message ?: "upload fail") }
            uploading = false
        }
    }

    fun submit() {
        scope.launch {
            submitting = true
            try {
                SurveyApi.report(id, lat, lng, note, suggestion, photoIds,
                    UUID.randomUUID().toString().replace("-", "").take(32))
                note = ""; photoIds = emptyList(); refresh++
                toast(ctx, ctx.getString(R.string.sv_submit_ok))
            } catch (e: Exception) { toast(ctx, ctx.getString(R.string.sv_submit_fail)) }
            submitting = false
        }
    }

    fun acceptTask() {
        scope.launch {
            try {
                SurveyApi.accept(id)
                toast(ctx, ctx.getString(R.string.sv_accept_ok))
                refresh++
            } catch (e: Exception) { toast(ctx, ctx.getString(R.string.sv_accept_fail)) }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.sv_detail_title), onBack = { nav.pop() }, action = stringResource(R.string.tool_action_refresh), onAction = { refresh++ })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(12.dp)) { Notice(stringResource(R.string.sv_load_fail), red = true) }
            is Load.Ok -> {
                val t = s.data.optJSONObject("task")
                val reports = s.data.optJSONArray("reports") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    if (t != null) {
                        KvRow(stringResource(R.string.sv_field_task), t.optString("taskNo") + " " + t.optString("title"))
                        if (t.optString("description").isNotEmpty()) KvRow(stringResource(R.string.sv_field_desc), t.optString("description"))
                        if (t.optLong("gridCode") > 0) KvRow(stringResource(R.string.sv_field_grid), t.optLong("gridCode").toString())
                        KvRow(stringResource(R.string.sv_field_status), svStatusText(t.optString("status")))
                        if (t.optString("status") == "PENDING") {
                            Spacer(Modifier.height(8.dp))
                            PrimaryButton(stringResource(R.string.sv_btn_accept), modifier = Modifier.fillMaxWidth()) { acceptTask() }
                        }
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.sv_reports) + " (" + reports.length() + ")")
                    if (reports.length() == 0) Empty(stringResource(R.string.sv_reports_empty))
                    for (i in 0 until reports.length()) {
                        val r = reports.optJSONObject(i) ?: continue
                        KvRow(r.optString("reportedAt"), svSuggestText(r.optString("suggestion")) +
                                if (r.optString("facilityNote").isNotEmpty()) " | " + r.optString("facilityNote") else "")
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.sv_backfill_title))
                    Row {
                        PrimaryButton(text = stringResource(R.string.sv_suggest_can), enabled = suggestion != "CAN_INSTALL", modifier = Modifier.weight(1f), onClick = { suggestion = "CAN_INSTALL" })
                        PrimaryButton(text = stringResource(R.string.sv_suggest_new), enabled = suggestion != "NEED_NEW_FACILITY", modifier = Modifier.weight(1f), onClick = { suggestion = "NEED_NEW_FACILITY" })
                    }
                    Text(if (lat != 0.0) ctx.getString(R.string.sv_located, lat.toString(), lng.toString()) else stringResource(R.string.sv_locating),
                        fontSize = 12.sp, color = Success, modifier = Modifier.padding(vertical = 6.dp));
                    PrimaryButton(text = stringResource(R.string.sv_btn_relocate), enabled = !locating) { locate() }
                    OutlinedTextField(value = note, onValueChange = { note = it },
                        label = { Text(stringResource(R.string.sv_note_hint)) }, modifier = Modifier.fillMaxWidth().padding(vertical = 6.dp))
                    PrimaryButton(text = stringResource(R.string.sv_btn_photo) + if (photoIds.isNotEmpty()) " (" + photoIds.size + ")" else "", enabled = !uploading, modifier = Modifier.fillMaxWidth()) {
                        picker.launch("image/*")
                    }
                    Spacer(Modifier.height(8.dp))
                    PrimaryButton(stringResource(R.string.sv_submit), enabled = !submitting, modifier = Modifier.fillMaxWidth()) { submit() }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}

fun svSuggestText(s: String): String = if (s == "NEED_NEW_FACILITY") "需新建设施" else "可装";