package com.ymm.boss.worker.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.ScanApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.launch

// 电子签收(对齐 docs/worker/sign.html):手写签名板 + 确认签收并激活
@Composable
fun SignScreen(nav: NavHost, no: String) {
    val charge by loadOnce(no) { ScanApi.charge(no) }
    var strokes by remember { mutableStateOf(listOf<List<Offset>>()) }
    var stroke by remember { mutableStateOf(listOf<Offset>()) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("电子签收", onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val c = charge) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice("收款信息加载失败：${c.message}", red = true)
                is Load.Ok -> {
                    KvRow("工单号", c.data.optString("ticketNo", no))
                    KvRow("装维结果", "四项确认完成", valueColor = Success)
                    KvRow("收款", "已到账 ¥${c.data.optDouble("amountDue", 0.0)}", valueColor = Success)
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            Text("请客户签字确认", fontSize = 15.sp, fontWeight = FontWeight.SemiBold,
                modifier = Modifier.fillMaxWidth().padding(vertical = 6.dp))
            Box(
                Modifier.fillMaxWidth().height(220.dp)
                    .background(Color.White).border(1.dp, Line)
                    .pointerInput(Unit) {
                        detectDragGestures(
                            onDragStart = { stroke = listOf(it) },
                            onDrag = { change, _ -> stroke = stroke + change.position },
                            onDragEnd = { strokes = strokes + listOf(stroke); stroke = emptyList() },
                            onDragCancel = { stroke = emptyList() },
                        )
                    },
            ) {
                Canvas(Modifier.fillMaxSize()) {
                    (strokes + listOf(stroke)).forEach { s ->
                        if (s.size > 1) {
                            val p = Path().apply { moveTo(s.first().x, s.first().y) }
                            s.drop(1).forEach { p.lineTo(it.x, it.y) }
                            drawPath(p, Ink, style = Stroke(4f, cap = StrokeCap.Round, join = StrokeJoin.Round))
                        }
                    }
                }
            }
            Spacer(Modifier.height(10.dp))
            PrimaryButton("清除重签", modifier = Modifier.fillMaxWidth()) {
                strokes = emptyList(); stroke = emptyList()
            }
            Spacer(Modifier.height(8.dp))
            PrimaryButton("确认签收并激活", modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    try {
                        val r = ScanApi.sign(no, "handwritten")
                        toast(ctx, r.optString("message", "签收成功"))
                        nav.switchTab(Screen.Orders)
                    } catch (e: Exception) { toast(ctx, "签收失败：${e.message}") }
                }
            }
        }
        androidx.compose.foundation.layout.Row(
            Modifier.fillMaxWidth().padding(horizontal = 12.dp),
            horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(10.dp),
        ) {
            PrimaryButton("现场收款", modifier = Modifier.weight(1f)) { nav.push(Screen.Charge(no)) }
            PrimaryButton("返回上报", modifier = Modifier.weight(1f)) { nav.push(Screen.Report(no)) }
        }
        Card(Modifier.padding(12.dp)) {
            Notice("签收回执留痕，作为工单完成与客户确认凭据；到付/现场收款见「现场收款」。")
        }
        Spacer(Modifier.height(12.dp))
    }
}