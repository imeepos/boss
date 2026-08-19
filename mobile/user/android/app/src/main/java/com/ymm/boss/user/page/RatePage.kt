package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch

// 对应草稿 docs/user/rate.html:服务评价?no=,星级 + 态度/质量打分 + 留言。
@Composable
fun RateScreen(nav: Nav, no: String) {
    var info by remember { mutableStateOf("") }
    var stars by remember { mutableIntStateOf(5) }
    var attitude by remember { mutableIntStateOf(5) }
    var quality by remember { mutableIntStateOf(5) }
    var comment by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    LaunchedEffect(no) {
        try {
            val d = OrderApi.rateInfo(no)
            info = "${d.optString("productName")} · ${d.optString("address")} · ${d.optString("finishedAt")} 完成"
        } catch (e: Exception) { info = "订单 $no" }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("服务评价", onBack = { nav.pop() })
        AppCard {
            CardTitle("订单 $no")
            Notice(info)
            StarRow("总体评分", stars) { stars = it }
            StarRow("装维师傅服务态度", attitude) { attitude = it }
            StarRow("装维质量与速度", quality) { quality = it }
            FieldLabel("评价留言(选填)")
            OutlinedTextField(
                value = comment, onValueChange = { comment = it },
                placeholder = { Text("分享你的装维体验") },
                minLines = 3, modifier = Modifier.fillMaxWidth(),
            )
            if (err.isNotEmpty()) Text(err, fontSize = 12.sp, color = Palette.err, modifier = Modifier.padding(top = 6.dp))
            SubmitButton(nav, no, stars, attitude, quality, comment) { err = it }
            Notice("评分低于 3 分将自动升级主管复核并回访。")
        }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun StarRow(label: String, value: Int, onChange: (Int) -> Unit) {
    Column(Modifier.padding(top = 10.dp)) {
        FieldLabel(label)
        Row {
            (1..5).forEach { i ->
                Text(
                    if (i <= value) "★" else "☆",
                    fontSize = 24.sp,
                    color = if (i <= value) Palette.warn else Palette.subtle,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.clickable { onChange(i) }.padding(end = 6.dp),
                )
            }
            Text(scoreLabel(value), fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(start = 6.dp, top = 8.dp))
        }
    }
}

private fun scoreLabel(v: Int) = when (v) {
    5 -> "5 分 · 非常满意"; 4 -> "4 分 · 满意"; 3 -> "3 分 · 一般"; 2 -> "2 分 · 不满意"; else -> "1 分 · 很不满意"
}

@Composable
private fun SubmitButton(
    nav: Nav, no: String, stars: Int, attitude: Int, quality: Int, comment: String,
    onErr: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    Button(
        onClick = {
            scope.launch {
                try {
                    OrderApi.submitRate(no, stars, attitude, quality, comment)
                    nav.pop()
                } catch (e: Exception) { onErr("提交失败,请稍后重试") }
            }
        },
        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
        modifier = Modifier.fillMaxWidth().padding(top = 14.dp).height(44.dp),
    ) { Text("提交评价") }
}
