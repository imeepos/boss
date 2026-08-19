package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

/** 对应草稿 docs/user/invoice.html:电子发票。GET/POST /invoices。 */
@Composable
fun InvoiceScreen(nav: Nav) {
    var d by remember { mutableStateOf<JSONObject?>(null) }

    LaunchedEffect(Unit) {
        try {
            d = BillApi.invoices()
        } catch (e: Exception) { /* 骨架保留默认值 */ }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("电子发票", onBack = { nav.pop() })
        InfoCard(d)
        AvailableCard(d)
        RecordsCard(d)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun InfoCard(d: JSONObject?) {
    AppCard {
        CardTitle("开票信息")
        CellRow(title = "抬头类型", right = { Value(d?.optString("titleType")) })
        CellRow(title = "发票抬头", right = { Value(d?.optString("title")) })
        val taxNo = d?.optString("taxNo") ?: ""
        CellRow(title = "税号", right = { Value(taxNo.ifBlank { "—" }) })
    }
}

@Composable
private fun AvailableCard(d: JSONObject?) {
    val scope = rememberCoroutineScope()
    var msg by remember { mutableStateOf("") }
    val periods = d?.optJSONArray("availablePeriods").optList()
    val issued = d?.optJSONArray("records").optList().map { it.optString("period") }.toSet()

    AppCard {
        CardTitle("可开票账期")
        if (msg.isNotEmpty()) Text(msg, fontSize = 12.5.sp, color = Palette.primary)
        if (periods.isEmpty()) {
            Text("暂无可开票账期", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
        }
        periods.forEach { b ->
            PeriodCell(b, b.optString("period") in issued) { billNo ->
                scope.launch {
                    try {
                        BillApi.applyInvoice(billNo)
                        msg = "已提交开票申请"
                    } catch (e: Exception) { msg = "开票申请失败,请重试" }
                }
            }
        }
    }
}

@Composable
private fun PeriodCell(b: JSONObject, issued: Boolean, onApply: (String) -> Unit) {
    val desc = "¥" + "%.2f".format(b.optDouble("amount")) + " · 已缴"
    if (issued) {
        CellRow(title = "${b.optString("period")} 账期", desc = desc,
            right = { Text("已开票", fontSize = 13.sp, color = Palette.muted) })
    } else {
        CellRow(title = "${b.optString("period")} 账期", desc = desc, right = {
            Button(onClick = { onApply(b.optString("billNo")) },
                colors = ButtonDefaults.buttonColors(containerColor = Palette.primary)) {
                Text("申请开票", fontSize = 12.5.sp)
            }
        })
    }
}

@Composable
private fun RecordsCard(d: JSONObject?) {
    val records = d?.optJSONArray("records").optList()
    AppCard {
        CardTitle("开票记录")
        if (records.isEmpty()) {
            Text("暂无开票记录", fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 8.dp))
        }
        records.forEach { r ->
            CellRow(
                title = "${r.optString("period")} 账期 · ¥" + "%.2f".format(r.optDouble("amount")),
                desc = "${r.optString("issuedAt")} 开票",
                right = {
                    Text("下载 PDF", fontSize = 13.sp, color = Palette.primary,
                        modifier = Modifier.clickable { /* TODO 下载地址 pdfUrl 交系统浏览器打开 */ })
                },
            )
        }
    }
}

@Composable
private fun Value(v: String?) {
    Text(v?.ifBlank { "—" } ?: "—", fontSize = 13.sp, color = Palette.ink)
}

private fun org.json.JSONArray?.optList(): List<JSONObject> {
    if (this == null) return emptyList()
    val out = ArrayList<JSONObject>(length())
    for (i in 0 until length()) optJSONObject(i)?.let { out.add(it) }
    return out
}
