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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.json.JSONArray
import com.ymm.boss.worker.R
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
        TopBar(stringResource(R.string.sign_title), onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            when (val c = charge) {
                is Load.Loading -> Loading()
                is Load.Fail -> Notice(stringResource(R.string.sign_load_fail, c.message), red = true)
                is Load.Ok -> {
                    KvRow(stringResource(R.string.td_ticket_no), c.data.optString("ticketNo", no))
                    KvRow(stringResource(R.string.td_title_done), stringResource(R.string.sign_result_complete), valueColor = Success)
                    KvRow(stringResource(R.string.sign_received), stringResource(R.string.sign_received_fmt, c.data.optDouble("amountDue", 0.0).toString()), valueColor = Success)
                }
            }
        }
        Card(Modifier.padding(12.dp)) {
            Text(stringResource(R.string.sign_hint), fontSize = 15.sp, fontWeight = FontWeight.SemiBold,
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
            PrimaryButton(stringResource(R.string.sign_clear), modifier = Modifier.fillMaxWidth()) {
                strokes = emptyList(); stroke = emptyList()
            }
            Spacer(Modifier.height(8.dp))
            PrimaryButton(stringResource(R.string.sign_confirm), enabled = strokes.isNotEmpty(), modifier = Modifier.fillMaxWidth()) {
                scope.launch {
                    try {
                        val signatureData = JSONArray(strokes.map { stroke ->
                            JSONArray(stroke.map { point -> JSONArray(listOf(point.x, point.y)) })
                        }).toString()
                        val r = ScanApi.sign(no, signatureData)
                        toast(ctx, r.optString("message", ctx.getString(R.string.sign_toast_ok)))
                        nav.switchTab(Screen.Orders)
                    } catch (e: Exception) { toast(ctx, ctx.getString(R.string.sign_toast_fail, e.message ?: "")) }
                }
            }
        }
        androidx.compose.foundation.layout.Row(
            Modifier.fillMaxWidth().padding(horizontal = 12.dp),
            horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(10.dp),
        ) {
            PrimaryButton(stringResource(R.string.sign_charge_btn), modifier = Modifier.weight(1f)) { nav.push(Screen.Charge(no)) }
            PrimaryButton(stringResource(R.string.sign_back_report), modifier = Modifier.weight(1f)) { nav.push(Screen.Report(no)) }
        }
        Card(Modifier.padding(12.dp)) {
            Notice(stringResource(R.string.sign_notice))
        }
        Spacer(Modifier.height(12.dp))
    }
}