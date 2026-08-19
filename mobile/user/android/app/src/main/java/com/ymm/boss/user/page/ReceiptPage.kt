package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/** 对应草稿 docs/user/receipt.html:缴费凭证。GET /payments/{payNo}/receipt。 */
@Composable
fun ReceiptScreen(nav: Nav, payNo: String) {
    var r by remember { mutableStateOf<JSONObject?>(null) }
    var failed by remember { mutableStateOf(false) }

    LaunchedEffect(payNo) {
        try {
            r = BillApi.receipt(payNo)
        } catch (e: Exception) { failed = true }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("缴费凭证", onBack = { nav.pop() }, action = "开发票") { nav.push(Route.Invoice) }
        ReceiptCard(r, payNo, failed)
        Button(
            onClick = { /* TODO 凭证 PDF 下载端点契约未提供 */ },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp).height(44.dp),
        ) { Text("下载凭证(PDF)") }
        Spacer(Modifier.height(8.dp))
    }
}

@Composable
private fun ReceiptCard(r: JSONObject?, payNo: String, failed: Boolean) {
    AppCard(Modifier.fillMaxWidth()) {
        Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            Text("电子缴费凭证", fontSize = 15.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
            Text(r?.optString("receiptNo")?.ifBlank { "—" } ?: "—", fontSize = 12.sp, color = Palette.muted)
        }
        if (failed) Notice("凭证加载失败,请稍后重试", Palette.err)
        CellRow(title = "客户", right = {
            Text(if (r != null) "${r.optString("customerName")} · ${r.optString("phoneMasked")}" else "—",
                fontSize = 13.sp, color = Palette.ink)
        })
        CellRow(title = "缴费金额", right = {
            Text(if (r != null) "¥" + "%.2f".format(r.optDouble("amount")) else "¥—",
                fontSize = 14.sp, fontWeight = FontWeight.Bold, color = Palette.primary)
        })
        ReceiptTail(r, payNo)
    }
}

@Composable
private fun ReceiptTail(r: JSONObject?, payNo: String) {
    CellRow(title = "对应账期", right = {
        Text(r?.optString("period")?.ifBlank { "—" } ?: "—", fontSize = 13.sp, color = Palette.ink)
    })
    CellRow(title = "支付方式 / 时间", right = {
        Text(
            if (r != null) BillApi.methodLabel(r.optString("payMethod")) + " · " + r.optString("paidAt") else "—",
            fontSize = 13.sp, color = Palette.ink,
        )
    })
    CellRow(title = "交易单号", right = {
        Text(r?.optString("payNo")?.ifBlank { payNo } ?: payNo, fontSize = 13.sp, color = Palette.ink)
    })
}
