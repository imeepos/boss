package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.util.devAutoFillSms
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// 找回密码(designs/login-register-states-v2 spec: 沿用登录卡片骨架与填充输入行)
// 端点: POST /auth/sms-code(scene=reset) + POST /auth/reset-password
@Composable
fun ForgotScreen(nav: Nav) {
    var phone by remember { mutableStateOf("13800001234") }
    var code by remember { mutableStateOf("") }
    var np by remember { mutableStateOf("") }
    var npVisible by remember { mutableStateOf(false) }
    var np2 by remember { mutableStateOf("") }
    var np2Visible by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    AuthCard("找回密码", "短信验证码重置 · 重置后需重新登录", onBack = { nav.pop() }) {
        AuthPhoneRow(phone, "请输入注册手机号") { phone = it }
        AuthCodeRow(code, countdown,
            onCode = { code = it },
            onSend = {
                sendResetCode(scope, phone, onCode = { code = it }) { err = it; countdown = 60 }
            })
        AuthPwdRow(np, { np = it }, "新密码(≥10位,含大小写与数字)", npVisible) { npVisible = !npVisible }
        AuthPwdRow(np2, { np2 = it }, "请再次输入新密码", np2Visible) { np2Visible = !np2Visible }
        if (err.isNotBlank()) Text(err, fontSize = 12.sp, color = com.ymm.boss.user.ui.Palette.err,
            modifier = Modifier.padding(top = 8.dp))
        RNPrimaryButton("确认重置", enabled = !busy, loading = busy,
            onClick = { busy = true; doReset(scope, nav, phone, code, np, np2) { err = it; busy = false } })
        Text("返回登录", fontSize = 12.sp, color = RN.primary, fontWeight = FontWeight.W500,
            modifier = Modifier.padding(top = 12.dp).clickable { nav.pop() })
    }
}

private fun sendResetCode(
    scope: kotlinx.coroutines.CoroutineScope,
    phone: String,
    onCode: (String) -> Unit,
    onDone: (String) -> Unit,
) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    if (!Regex("^1\\d{10}$").matches(phone)) { onDone("手机号格式不正确"); return }
    scope.launch {
        try {
            UserApi.auth.smsCode(phone, "reset")
            devAutoFillSms(phone, "reset", onCode)
            onDone("")
        } catch (e: Exception) { onDone("验证码发送失败：" + Api.friendlyMessage(e)) }
    }
}

private fun doReset(
    scope: kotlinx.coroutines.CoroutineScope, nav: Nav,
    phone: String, code: String, np: String, np2: String,
    onErr: (String) -> Unit,
) {
    if (phone.isBlank() || code.isBlank() || np.isBlank()) { onErr("请填写完整"); return }
    if (np != np2) { onErr("两次密码不一致"); return }
    if (np.length < 10 || np.none { it.isDigit() } || np.none { it.isLowerCase() } || np.none { it.isUpperCase() }) {
        onErr("密码需≥10 位且含大小写与数字"); return
    }
    scope.launch {
        try {
            AccountApi.resetPassword(phone, code, np)
            nav.replace(Route.Login) // 重置成功跳回登录页
        } catch (e: Exception) { onErr("重置失败：" + Api.friendlyMessage(e)) }
    }
}
