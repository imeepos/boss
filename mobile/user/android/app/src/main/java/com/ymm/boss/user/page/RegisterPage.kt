package com.ymm.boss.user.page

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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar
import com.ymm.boss.user.ui.theme.RnPalette
import com.ymm.boss.user.util.devAutoFillSms
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// 注册页(designs/login-register-states-v2 spec: 沿用登录卡片骨架与填充输入行)
// 端点: POST /auth/sms-code(scene=register) + POST /auth/register
@Composable
fun RegisterScreen(nav: Nav) {
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var pwd by remember { mutableStateOf("") }
    var pwd2 by remember { mutableStateOf("") }
    var pwdVisible by remember { mutableStateOf(false) }
    var pwd2Visible by remember { mutableStateOf(false) }
    var agreed by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    AuthCard("自助注册", "手机号注册 · 注册即享宽带自助服务", onBack = { nav.pop() }) {
        AuthPhoneRow(phone, "请输入手机号") { phone = it }
        AuthCodeRow(code, countdown,
            onCode = { code = it },
            onSend = {
                sendRegisterCode(scope, phone, onCode = { code = it }) { err = it; countdown = 59 }
            })
        AuthPwdRow(pwd, { pwd = it }, "设置密码(≥10位,含大小写与数字)", pwdVisible) { pwdVisible = !pwdVisible }
        AuthPwdRow(pwd2, { pwd2 = it }, "请再次输入密码", pwd2Visible) { pwd2Visible = !pwd2Visible }
        if (err.isNotBlank()) Text(err, fontSize = 12.sp, color = Palette.err,
            modifier = Modifier.padding(top = 8.dp))
        AuthAgreeRow(agreed, { agreed = it }) { nav.push(Route.Agreement) }
        RNPrimaryButton("注册并登录", enabled = agreed && !busy, loading = busy,
            onClick = {
                busy = true
                doRegister(scope, nav, phone, code, pwd, pwd2) { err = it; busy = false }
            })
        Text("已有账号，去登录", fontSize = 12.sp, color = RnPalette.primary, fontWeight = FontWeight.W500,
            modifier = Modifier.padding(top = 12.dp).clickable { nav.pop() })
    }
}

// 内部校验: 必填/两次一致,另按标签约定校验密码强度
private fun pwdInvalid(p: String): Boolean =
    p.length < 10 || p.none { it.isDigit() } || p.none { it.isLowerCase() } || p.none { it.isUpperCase() }

private fun sendRegisterCode(
    scope: kotlinx.coroutines.CoroutineScope,
    phone: String,
    onCode: (String) -> Unit,
    onDone: (String) -> Unit,
) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    if (!Regex("^1\\d{10}$").matches(phone)) { onDone("手机号格式不正确"); return }
    scope.launch {
        try {
            UserApi.auth.smsCode(phone, "register")
            devAutoFillSms(phone, "register", onCode)
            onDone("")
        } catch (e: Exception) { onDone("验证码发送失败：" + com.ymm.boss.user.api.Api.friendlyMessage(e)) }
    }
}

private fun doRegister(
    scope: kotlinx.coroutines.CoroutineScope, nav: Nav,
    phone: String, code: String, pwd: String, pwd2: String,
    onErr: (String) -> Unit,
) {
    if (phone.isBlank() || code.isBlank() || pwd.isBlank()) { onErr("请填写完整"); return }
    if (pwd != pwd2) { onErr("两次密码不一致"); return }
    if (pwdInvalid(pwd)) { onErr("密码需≥10 位且含大小写与数字"); return }
    scope.launch {
        try {
            UserApi.auth.register(phone, code, pwd)
            // 注册申请需后台审核(契约 fields.md 7.6),成功后回登录页
            nav.replace(Route.Login)
        } catch (e: Exception) { onErr("注册失败：" + com.ymm.boss.user.api.Api.friendlyMessage(e)) }
    }
}

/** 注册/找回共用骨架:TopBar + 居中白卡(10dp 圆角 + 16dp 内边距)。 */
@Composable
internal fun AuthCard(title: String, sub: String, onBack: (() -> Unit)? = null, content: @Composable () -> Unit) {
    Column(Modifier.fillMaxSize().background(Palette.bg)) {
        if (onBack != null) TopBar(title, onBack = onBack)
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(20.dp))
            Column(
                Modifier.background(Color.White, RoundedCornerShape(10.dp)).padding(16.dp).fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(title, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
                Text(sub, fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 6.dp, bottom = 8.dp))
                content()
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}
