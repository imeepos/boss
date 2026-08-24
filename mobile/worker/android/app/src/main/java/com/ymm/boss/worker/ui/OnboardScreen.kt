package com.ymm.boss.worker.ui

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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.outlined.Badge
import androidx.compose.material.icons.outlined.Group
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.PhoneAndroid
import androidx.compose.material.icons.outlined.Shield
import androidx.compose.material3.Icon
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
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.api.friendlyMessage
import com.ymm.boss.worker.api.AuthApi
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import com.ymm.boss.worker.ui.theme.Success
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// 申请入驻页面:姓名/手机/验证码/身份证/班组ID/区域ID(可留0待后台补正)。
// 提交后展示"已提交审核"状态页。

@Composable
fun OnboardScreen(onBack: () -> Unit, onAgreement: () -> Unit = {}) {
    var submitted by remember { mutableStateOf(false) }
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var idCardNo by remember { mutableStateOf("") }
    var groupId by remember { mutableStateOf("") }
    var regionId by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    if (submitted) {
        OnboardSuccess(onBack)
        return
    }

    Column(
        Modifier.fillMaxSize().background(Color(0xFFF6F8FA))
            .verticalScroll(rememberScrollState()),
    ) {
        // 顶部渐变标题栏
        OnboardHeader(onBack)

        Column(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 16.dp),
        ) {
            // 白色卡片:表单
            Column(
                Modifier.fillMaxWidth()
                    .background(Color.White, RoundedCornerShape(10.dp))
                    .padding(16.dp),
            ) {
                Text("申请入驻", fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Ink)
                Text("提交后需等待后台审核,审核通过方可登录", fontSize = 12.sp, color = Muted,
                    modifier = Modifier.padding(top = 6.dp, bottom = 12.dp))

                AuthInputRow(name, { name = it }, "请输入姓名", Icons.Outlined.Person, KeyboardType.Text)
                AuthPhoneRow(phone, "请输入手机号") { phone = it }
                AuthCodeRow(code, countdown,
                    onCode = { code = it },
                    onSend = {
                        sendOnboardCode(scope, phone) { err = it; countdown = 59 }
                    })
                AuthInputRow(idCardNo, { idCardNo = it }, "请输入身份证号", Icons.Outlined.Badge, KeyboardType.Text)
                // 班组/区域 ID 可留空,0 表示待后台补正
                AuthInputRow(groupId, { groupId = it }, "班组 ID(选填)", Icons.Outlined.Group, KeyboardType.Number)
                AuthInputRow(regionId, { regionId = it }, "区域 ID(选填)", Icons.Outlined.LocationOn, KeyboardType.Number)
                if (err.isNotBlank()) AuthFootnote(err, Color(0xFFFF2D2F))
            }

            // 协议行
            AuthAgreeRow(agreed, onToggle = { agreed = it }, onAgreement = onAgreement)

            // 提交按钮
            AuthPrimaryButton("提交入驻申请", enabled = agreed && !busy, loading = busy,
                onClick = {
                    busy = true
                    doOnboard(scope, name, phone, code, idCardNo, groupId, regionId) { ok, msg ->
                        busy = false
                        if (ok) submitted = true else err = msg
                    }
                })

            // 返回登录
            Row(
                Modifier.fillMaxWidth().padding(vertical = 16.dp),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("已有账号？", fontSize = 12.sp, color = Muted)
                Text("去登录", fontSize = 12.sp, color = Primary, fontWeight = FontWeight.W500,
                    modifier = Modifier.clickable { onBack() })
            }
        }
    }
}

// 顶部渐变标题栏
@Composable
private fun OnboardHeader(onBack: () -> Unit) {
    Box(
        Modifier.fillMaxWidth().background(
            Brush.linearGradient(listOf(Primary, Primary2)))
            .padding(horizontal = 16.dp, vertical = 20.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("<", color = Color.White, fontSize = 16.sp,
                modifier = Modifier.clickable { onBack() }.padding(4.dp))
            Text("申请入驻", color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(start = 12.dp))
        }
    }
}

// 提交成功状态页
@Composable
private fun OnboardSuccess(onBack: () -> Unit) {
    Column(
        Modifier.fillMaxSize().background(Color(0xFFF6F8FA))
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        // 绿色勾选图标
        Box(
            Modifier.size(72.dp).background(Success.copy(alpha = 0.1f), CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Filled.CheckCircle, contentDescription = null,
                tint = Success, modifier = Modifier.size(48.dp))
        }

        Spacer(Modifier.height(24.dp))

        Text("已提交审核", fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Ink)
        Text("您的入驻申请已提交,请耐心等待后台审核", fontSize = 14.sp, color = Muted,
            modifier = Modifier.padding(top = 8.dp))
        Text("审核期间请勿重复提交", fontSize = 12.sp, color = Muted,
            modifier = Modifier.padding(top = 4.dp))

        Spacer(Modifier.height(32.dp))

        AuthPrimaryButton("返回登录", enabled = true, onClick = onBack)
    }
}

private fun sendOnboardCode(
    scope: kotlinx.coroutines.CoroutineScope,
    phone: String,
    onDone: (String) -> Unit,
) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    if (!Regex("^1\\d{10}$").matches(phone)) { onDone("手机号格式不正确"); return }
    scope.launch {
        try {
            AuthApi.smsCode(phone.trim())
            onDone("")
        } catch (e: Exception) { onDone("验证码发送失败：${friendlyMessage(e)}") }
    }
}

private fun doOnboard(
    scope: kotlinx.coroutines.CoroutineScope,
    name: String, phone: String, code: String, idCardNo: String,
    groupId: String, regionId: String,
    onDone: (Boolean, String) -> Unit,
) {
    if (name.isBlank() || phone.isBlank() || code.isBlank() || idCardNo.isBlank()) {
        onDone(false, "请填写完整信息"); return
    }
    val gId = groupId.toLongOrNull() ?: 0
    val rId = regionId.toLongOrNull() ?: 0
    scope.launch {
        try {
            val body = org.json.JSONObject().apply {
                put("name", name.trim())
                put("phone", phone.trim())
                put("smsCode", code.trim())
                put("idCardNo", idCardNo.trim())
                put("groupId", gId)
                put("regionId", rId)
            }
            Api.post("/worker-registrations", body)
            onDone(true, "")
        } catch (e: Exception) {
            onDone(false, "提交失败：${friendlyMessage(e)}")
        }
    }
}
