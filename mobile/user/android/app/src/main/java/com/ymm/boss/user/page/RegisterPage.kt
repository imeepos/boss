package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Checkbox
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// 对应草稿 docs/user/register.html:手机号+验证码+密码自助注册
// 端点: POST /auth/sms-code(scene=register) + POST /auth/register
@Composable
fun RegisterScreen(nav: Nav) {
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var pwd by remember { mutableStateOf("") }
    var pwd2 by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(true) }
    var err by remember { mutableStateOf("") }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    AuthCard("自助注册", "手机号注册 · 注册即享宽带自助服务", onBack = { nav.pop() }) {
        OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("手机号") },
            placeholder = { Text("请输入手机号") }, singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            modifier = Modifier.fillMaxWidth().padding(top = 4.dp))
        CodeField(code, countdown,
            onCode = { code = it },
            onSend = { sendRegisterCode(scope, phone) { err = it; countdown = 59 } })
        PwdField("设置密码(≥10 位,含大小写与数字)", pwd, { pwd = it }, "请设置登录密码")
        PwdField("确认密码", pwd2, { pwd2 = it }, "请再次输入密码")
        AgreeRow(agreed, { agreed = it }) { nav.push(Route.Agreement) }
        RegisterFooter(err,
            onSubmit = { doRegister(scope, nav, phone, code, pwd, pwd2, agreed) { err = it } },
            onLogin = { nav.pop() })
    }
}

@Composable
private fun RegisterFooter(err: String, onSubmit: () -> Unit, onLogin: () -> Unit) {
    Text(err, fontSize = 12.sp, color = Palette.err, modifier = Modifier.padding(bottom = 6.dp, top = 2.dp))
    RNPrimaryButton("注册并登录", enabled = true, onClick = onSubmit)
    Text("已有账号，去登录", fontSize = 12.sp, color = RN.primary,
        modifier = Modifier.padding(top = 12.dp).clickable { onLogin() })
}

// 内部校验: 与草稿一致(必填/两次一致),另按标签约定校验密码强度
private fun pwdInvalid(p: String): Boolean =
    p.length < 10 || p.none { it.isDigit() } || p.none { it.isLowerCase() } || p.none { it.isUpperCase() }

private fun sendRegisterCode(scope: kotlinx.coroutines.CoroutineScope, phone: String, onDone: (String) -> Unit) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    scope.launch {
        try {
            UserApi.auth.smsCode(phone, "register"); onDone("验证码已发送")
        } catch (e: Exception) { onDone("验证码发送失败") }
    }
}

private fun doRegister(
    scope: kotlinx.coroutines.CoroutineScope, nav: Nav,
    phone: String, code: String, pwd: String, pwd2: String,
    agreed: Boolean, onErr: (String) -> Unit,
) {
    if (phone.isBlank() || code.isBlank() || pwd.isBlank()) { onErr("请填写完整"); return }
    if (pwd != pwd2) { onErr("两次密码不一致"); return }
    if (pwdInvalid(pwd)) { onErr("密码需≥10 位且含大小写与数字"); return }
    if (!agreed) { onErr("请先同意用户协议"); return }
    scope.launch {
        try {
            UserApi.auth.register(phone, code, pwd)
            // 草稿为 setToken 后跳首页;注册申请需后台审核(契约 fields.md 7.6),
            // 故按任务要求成功后回登录页
            nav.replace(Route.Login)
        } catch (e: Exception) { onErr("注册失败") }
    }
}

// 以下为注册/找回两页共用的私有组件,各自文件内声明,避免改动共享 ui 包
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
                // login-register-states-v2 骨架:卡片 10dp 圆角 + 16dp 内边距
                Modifier.background(Palette.panel, RoundedCornerShape(10.dp)).padding(16.dp).fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(title, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Palette.ink)
                Text(sub, fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 6.dp, bottom = 20.dp))
                content()
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
internal fun CodeField(code: String, countdown: Int, onCode: (String) -> Unit, onSend: () -> Unit) {
    OutlinedTextField(
        value = code, onValueChange = onCode, label = { Text("验证码") },
        placeholder = { Text("6 位验证码") }, singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        trailingIcon = {
            Text(
                if (countdown > 0) "${countdown}s后重发" else "获取验证码",
                fontSize = 12.sp, fontWeight = FontWeight.W500,
                color = if (countdown > 0) RN.placeholder else RN.primary,
                modifier = Modifier.padding(end = 8.dp).clickable(enabled = countdown == 0) { onSend() },
            )
        },
    )
}

@Composable
internal fun PwdField(label: String, value: String, onChange: (String) -> Unit, hint: String) {
    OutlinedTextField(value = value, onValueChange = onChange, label = { Text(label) },
        placeholder = { Text(hint) }, singleLine = true,
        visualTransformation = PasswordVisualTransformation(),
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
}

@Composable
private fun AgreeRow(agreed: Boolean, onChange: (Boolean) -> Unit, onOpenAgreement: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(top = 10.dp), verticalAlignment = Alignment.CenterVertically) {
        Checkbox(checked = agreed, onCheckedChange = onChange)
        Text("已阅读并同意", fontSize = 12.5.sp, color = Palette.muted)
        Text("《用户协议》", fontSize = 12.5.sp, color = Palette.primary, modifier = Modifier.clickable { onOpenAgreement() })
    }
}
