package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.statusBarSolid
import org.json.JSONObject

// 实名认证分步流程(designs/realname-flow-states-v1):3 步 + 4 状态页。
// 端点: GET /auth/verify + POST /auth/verify/sms-code + POST /auth/verify + POST /attachments/upload

/** 实名流程专属 token(designs/realname-flow-states-v1.spec.md;与全局 Palette 同族)。 */
internal object RN {
    val primary = Color(0xFF086CF5)
    val heroEnd = Color(0xFF1698FA)
    val success = Color(0xFF0AA847)
    val successBg = Color(0xFFEFFFF4)
    val warn = Color(0xFFF57900)
    val warnBg = Color(0xFFFFF5E8)
    val ink = Color(0xFF171B23)
    val muted = Color(0xFF5F6671)
    val placeholder = Color(0xFFAEB4BE)
    val line = Color(0xFFE7EAF0)
}

/** 流程阶段:两步表单 + 三个互斥状态页(verifyStatus/latestResult 条件渲染)。 */
internal enum class RNPhase { FORM, UPLOAD, REVIEWING, APPROVED, REJECTED }

/** GET /auth/verify 载荷 → 初始阶段(从未提交进表单,最新记录判定状态页)。 */
internal fun initialPhase(d: JSONObject): RNPhase = when {
    d.optString("status") == "VERIFIED" -> RNPhase.APPROVED
    d.optString("latestResult") == "FAIL" -> RNPhase.REJECTED
    d.optString("latestResult") == "PENDING" -> RNPhase.REVIEWING
    else -> RNPhase.FORM
}

/** phase → stepper 当前步(1 填写信息 / 2 证件上传 / 3 审核状态)。 */
internal fun phaseStep(p: RNPhase): Int = when (p) {
    RNPhase.FORM -> 1; RNPhase.UPLOAD -> 2; else -> 3
}

@Composable
fun VerifyScreen(nav: com.ymm.boss.user.ui.Nav) {
    var data by remember { mutableStateOf<JSONObject?>(null) }
    var loading by remember { mutableStateOf(true) }
    var phase by rememberSaveable { mutableStateOf(RNPhase.FORM) }
    // 表单与附件状态提升:UPLOAD 步提交时要带上 FORM 步资料
    var name by rememberSaveable { mutableStateOf("") }
    var idNo by rememberSaveable { mutableStateOf("") }
    var frontId by rememberSaveable { mutableStateOf(0L) }
    var backId by rememberSaveable { mutableStateOf(0L) }

    suspend fun reload() {
        try {
            val d = AccountApi.verifyStatus()
            data = d
            phase = initialPhase(d)
        } catch (e: Exception) { data = null } // 拉取失败不白屏,按待填表单展示
        finally { loading = false }
    }
    LaunchedEffect(nav.refreshTick) { reload() }

    Column(Modifier.fillMaxSize()) {
        com.ymm.boss.user.ui.TopBar("实名认证", onBack = { nav.pop() })
        RNStepper(current = phaseStep(phase))
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp),
        ) {
            Spacer(Modifier.height(8.dp))
            when (phase) {
                RNPhase.FORM -> RNFormStep(data, name, idNo,
                    onName = { name = it }, onIdNo = { idNo = it },
                    onNext = { phase = RNPhase.UPLOAD })
                RNPhase.UPLOAD -> RNUploadStep(name, idNo, frontId, backId,
                    onFront = { frontId = it }, onBack_ = { backId = it },
                    onSubmitted = { phase = it })
                RNPhase.REVIEWING -> RNReviewingPage(data)
                RNPhase.APPROVED -> RNApprovedPage(data)
                RNPhase.REJECTED -> RNRejectedPage(data, onResubmit = {
                    frontId = 0; backId = 0; phase = RNPhase.FORM
                })
            }
            if (loading) RNFootnote("加载中…")
            Spacer(Modifier.height(24.dp))
        }
    }
}

/** 三步 Stepper:当前实心蓝,已完成蓝+对勾,未到灰(spec: 单一样式组件,四屏复用)。 */
@Composable
private fun RNStepper(current: Int) {
    val labels = listOf("填写信息", "证件上传", "审核状态")
    Row(
        Modifier.fillMaxWidth().background(Palette.bg).padding(vertical = 12.dp),
        horizontalArrangement = Arrangement.Center,
    ) {
        labels.forEachIndexed { i, label ->
            if (i > 0) Box(Modifier.width(28.dp).height(1.dp).align(Alignment.CenterVertically)
                .background(if (i < current) RN.primary else RN.line))
            RNStepNode(i + 1, label, i + 1 == current, i + 1 < current)
        }
    }
}

@Composable
private fun RNStepNode(step: Int, label: String, current: Boolean, done: Boolean) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.size(18.dp).background(
                when { done -> RN.primary; current -> RN.primary; else -> RN.placeholder },
                CircleShape,
            ), contentAlignment = Alignment.Center,
        ) {
            if (done) androidx.compose.material3.Icon(
                Icons.Filled.Check, contentDescription = null,
                tint = Color.White, modifier = Modifier.size(12.dp))
            else androidx.compose.material3.Text(
                "$step", fontSize = 10.sp, color = Color.White, fontWeight = FontWeight.W600)
        }
        Spacer(Modifier.width(6.dp))
        androidx.compose.material3.Text(
            label, fontSize = 12.sp,
            color = if (current || done) RN.ink else RN.muted,
            fontWeight = if (current) FontWeight.W600 else FontWeight.Normal)
    }
}

/** 主操作按钮:≥44dp,禁用灰,loading 转圈。 */
@Composable
internal fun RNPrimaryButton(text: String, enabled: Boolean, loading: Boolean = false, onClick: () -> Unit) {
    val bg = if (enabled) RN.primary else RN.placeholder
    Box(
        Modifier.fillMaxWidth().padding(vertical = 4.dp).height(46.dp)
            .background(bg, RoundedCornerShape(10.dp))
            .clickable(enabled = enabled && !loading) { onClick() },
        contentAlignment = Alignment.Center,
    ) {
        if (loading) androidx.compose.material3.CircularProgressIndicator(
            Modifier.size(20.dp), color = Color.White, strokeWidth = 2.dp)
        else androidx.compose.material3.Text(text, fontSize = 15.sp, color = Color.White, fontWeight = FontWeight.W600)
    }
}

/** 辅助/说明小字。 */
@Composable
internal fun RNFootnote(text: String, color: Color = RN.muted) {
    androidx.compose.material3.Text(
        text, fontSize = 12.sp, color = color, modifier = Modifier.padding(vertical = 4.dp))
}
