package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.AssetApi
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.launch
import org.json.JSONArray

// 领料/借还(对齐 docs/worker/pickup.html):今日领用清单 + 工具借还
@Composable
fun PickupScreen(nav: NavHost) {
    var refresh by remember { mutableStateOf(0) }
    val materials by loadOnce(refresh) { AssetApi.materials() }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val refreshLabel = stringResource(R.string.tool_action_refresh)
    val outOk = stringResource(R.string.pick_out_ok)
    val borrowOk = stringResource(R.string.pick_borrow_ok)
    val returnOk = stringResource(R.string.pick_return_ok)

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.pick_title), onBack = { nav.pop() }, action = refreshLabel, onAction = { refresh++ })
        when (val m = materials) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(ctx.getString(R.string.pick_load_fail, m.message ?: ""), red = true) }
            is Load.Ok -> {
                val items = m.data.optJSONArray("items") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.pick_section_today), more = ctx.getString(R.string.pick_count_items, items.length()))
                    if (items.length() == 0) Empty(stringResource(R.string.pick_empty_today))
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        val id = it0.optString("itemId")
                        val out = it0.optBoolean("outBound")
                        Cell(
                            title = "${it0.optString("name")} ×${it0.optInt("qty")}",
                            desc = it0.optString("spec"),
                        ) {
                            Text(if (out) stringResource(R.string.pick_status_out) else stringResource(R.string.pick_btn_out), fontSize = 12.sp,
                                color = if (out) Muted else Color.White,
                                modifier = Modifier
                                    .clip(RoundedCornerShape(6.dp))
                                    .background(if (out) Color.Transparent else Primary)
                                    .clickable(enabled = !out) {
                                        scope.launch {
                                            try {
                                                val r = AssetApi.materialOut(id)
                                                toast(ctx, r.optString("message", outOk))
                                                refresh++
                                            } catch (e: Exception) { toast(ctx, ctx.getString(R.string.pick_out_fail, e.message ?: "")) }
                                        }
                                    }
                                    .padding(horizontal = 10.dp, vertical = 5.dp))
                        }
                    }
                }
                val toolsState by loadOnce(refresh) { AssetApi.tools() }
                val tools = (toolsState as? Load.Ok)?.data?.optJSONArray("items") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    SectionTitle(stringResource(R.string.pick_section_tools), more = ctx.getString(R.string.pick_count_tools, tools.length()))
                    if (tools.length() == 0) Empty(stringResource(R.string.pick_empty_tools))
                    for (i in 0 until tools.length()) {
                        val t = tools.optJSONObject(i)
                        val id = t.optString("toolId")
                        val borrowed = t.optBoolean("borrowed")
                        Cell(title = t.optString("name")) {
                            Text(if (borrowed) stringResource(R.string.pick_btn_return) else stringResource(R.string.pick_btn_borrow), fontSize = 12.sp, color = Color.White,
                                modifier = Modifier
                                    .clip(RoundedCornerShape(6.dp))
                                    .background(if (borrowed) Muted else Primary)
                                    .clickable {
                                        scope.launch {
                                            try {
                                                val r = if (borrowed) AssetApi.giveBackTool(id)
                                                else AssetApi.borrowTool(id)
                                                toast(ctx, r.optString("message",
                                                    if (borrowed) returnOk else borrowOk))
                                                refresh++
                                            } catch (e: Exception) { toast(ctx, ctx.getString(R.string.pick_op_fail, e.message ?: "")) }
                                        }
                                    }
                                    .padding(horizontal = 10.dp, vertical = 5.dp))
                        }
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    Notice(stringResource(R.string.pick_inventory_notice))
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}