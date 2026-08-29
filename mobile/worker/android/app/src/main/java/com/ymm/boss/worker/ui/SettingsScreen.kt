package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

// 接单设置(对齐 docs/worker/settings.html):在线状态 + 接单类型
@Composable
fun SettingsScreen(nav: NavHost) {
    val settings by loadOnce { ProfileApi.settings() }
    val ctx = LocalContext.current
    val typesNewInstall = stringResource(R.string.settings_type_new_install)
    val typesChange = stringResource(R.string.settings_type_change)
    val typesDismantle = stringResource(R.string.settings_type_dismantle)
    val typesRepair = stringResource(R.string.settings_type_repair)
    val acceptTypes = listOf(typesNewInstall, typesChange, typesDismantle, typesRepair)
    val toastOk = stringResource(R.string.settings_save_ok)
    var online by remember { mutableStateOf(true) }
    var radiusKm by remember { mutableStateOf(5) }
    var types by remember { mutableStateOf(setOf<String>()) }
    var tip by remember { mutableStateOf("") }
    var saving by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(settings) {
        val d = (settings as? Load.Ok)?.data ?: return@LaunchedEffect
        online = d.optBoolean("online", true)
        radiusKm = d.optInt("radiusKm", 5)
        types = (0 until (d.optJSONArray("acceptTypes")?.length() ?: 0))
            .map { d.optJSONArray("acceptTypes")!!.optString(it) }.toSet()
    }

    fun save() {
        if (saving || settings !is Load.Ok) return
        saving = true
        scope.launch {
            tip = try {
                val list = JSONArray()
                acceptTypes.filter { it in types }.forEach { list.put(it) }
                val r = ProfileApi.saveSettings(JSONObject().apply {
                    put("online", online); put("radiusKm", radiusKm); put("acceptTypes", list)
                })
                toast(ctx, r.optString("message", toastOk))
                ""
            } catch (e: Exception) { ctx.getString(R.string.settings_save_fail, e.message ?: "") }
            saving = false
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.settings_title), onBack = { nav.pop() }, action = stringResource(R.string.settings_save), onAction = { save() })
        Card(Modifier.padding(12.dp)) {
            Cell(stringResource(R.string.settings_status_label), onClick = { online = !online }) {
                Text(if (online) stringResource(R.string.settings_status_on) else stringResource(R.string.settings_status_off), fontSize = 13.sp,
                    color = Color.White,
                    modifier = Modifier
                        .background(if (online) Primary else Muted, RoundedCornerShape(6.dp))
                        .padding(horizontal = 10.dp, vertical = 5.dp))
            }
            KvRow(stringResource(R.string.settings_kv_radius), "$radiusKm km")
            KvRow(stringResource(R.string.settings_kv_types), if (types.isEmpty()) stringResource(R.string.settings_radius_unset) else types.joinToString(" / "))
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle(stringResource(R.string.settings_kv_types))
            acceptTypes.forEach { t ->
                val on = t in types
                Row(Modifier.fillMaxWidth().clickable {
                    types = if (on) types - t else types + t
                }.padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text(if (on) "☑" else "☐", fontSize = 16.sp,
                        color = if (on) Primary else Color(0xFFB8BCC2))
                    Text(t, fontSize = 14.sp, modifier = Modifier.padding(start = 10.dp))
                }
            }
        }
        if (tip.isNotEmpty()) Notice(tip, red = true)
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.settings_pause_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}