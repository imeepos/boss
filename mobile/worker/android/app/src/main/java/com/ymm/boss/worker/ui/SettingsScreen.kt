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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.ProfileApi
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

private val ACCEPT_TYPES = listOf("新装宽带", "宽带变更", "拆机", "抢修（需资质）")

// 接单设置(对齐 docs/worker/settings.html):在线状态 + 接单类型
@Composable
fun SettingsScreen(nav: NavHost) {
    val settings by loadOnce { ProfileApi.settings() }
    var online by remember { mutableStateOf(true) }
    var types by remember { mutableStateOf(setOf<String>()) }
    var tip by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    LaunchedEffect(settings) {
        val d = (settings as? Load.Ok)?.data ?: return@LaunchedEffect
        online = d.optBoolean("online", true)
        types = (0 until (d.optJSONArray("acceptTypes")?.length() ?: 0))
            .map { d.optJSONArray("acceptTypes")!!.optString(it) }.toSet()
    }

    fun save() {
        scope.launch {
            tip = try {
                val list = JSONArray()
                ACCEPT_TYPES.filter { it in types }.forEach { list.put(it) }
                val r = ProfileApi.saveSettings(JSONObject().apply {
                    put("online", online); put("radiusKm", 5); put("acceptTypes", list)
                })
                toast(ctx, r.optString("message", "接单设置已保存"))
                ""
            } catch (e: Exception) { "保存失败：${e.message}" }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("接单设置", onBack = { nav.pop() }, action = "保存", onAction = { save() })
        Card(Modifier.padding(12.dp)) {
            Cell("接单状态", onClick = { online = !online }) {
                Text(if (online) "在线接单" else "已停接单", fontSize = 13.sp,
                    color = Color.White,
                    modifier = Modifier
                        .background(if (online) Primary else Color(0xFF8C8C8C), RoundedCornerShape(6.dp))
                        .padding(horizontal = 10.dp, vertical = 5.dp))
            }
            KvRow("接单半径", "5 km")
            KvRow("接单类型", if (types.isEmpty()) "未设置" else types.joinToString(" / "))
        }
        Card(Modifier.padding(12.dp)) {
            SectionTitle("能接单类型")
            ACCEPT_TYPES.forEach { t ->
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
            Notice("停止接单后不再派发新工单，进行中工单不受影响。")
        }
        Spacer(Modifier.height(12.dp))
    }
}