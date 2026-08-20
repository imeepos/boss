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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import kotlinx.coroutines.launch
import org.json.JSONArray

// 旧件回收(对齐 docs/worker/retire.html):待回收旧件列表 + 返库
@Composable
fun RetireScreen(nav: NavHost, no: String) {
    var refresh by remember { mutableStateOf(0) }
    val state by loadOnce(refresh) { AssetApi.materials() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("旧件回收", onBack = { nav.pop() })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("加载失败：${s.message}", red = true) }
            is Load.Ok -> {
                val items = s.data.optJSONArray("pendingReturn") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle("待回收旧件")
                    if (items.length() == 0) Empty("暂无待回收旧件")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        Row(Modifier.fillMaxWidth().padding(vertical = 8.dp),
                            verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text("光猫 ${it0.optString("epc")}", fontSize = 14.sp, color = Ink)
                                Text(it0.optString("reason"), fontSize = 12.sp, color = Muted)
                            }
                            Text("返库", fontSize = 13.sp, color = Color.White,
                                modifier = Modifier
                                    .clip(RoundedCornerShape(8.dp))
                                    .background(Primary)
                                    .clickable {
                                        scope.launch {
                                            try {
                                                val r = AssetApi.returnAsset(it0.optString("epc"))
                                                toast(ctx, r.optString("message", "已登记返库"))
                                                refresh++
                                            } catch (e: Exception) { toast(ctx, "返库失败：${e.message}") }
                                        }
                                    }
                                    .padding(horizontal = 12.dp, vertical = 6.dp))
                        }
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    SectionTitle("已回收记录", more = "本月")
                    val returned = s.data.optJSONObject("returned")
                    KvRow("返修件", "${returned?.optInt("repairCount", 0) ?: 0} 件")
                    KvRow("拆机回收", "${returned?.optInt("dismantleCount", 0) ?: 0} 件")
                }
                Card(Modifier.padding(12.dp)) {
                    Notice("旧件回收须扫码留痕，返修件与拆机件分流处理。")
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}