package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// 对应草稿 docs/user/forgot.html:短信验证码重置密码
// 端点: POST /auth/sms-code(scene=reset) + POST /auth/reset-password
@Composable
fun ForgotScreen(nav: Nav) {
    var phone by remember { mutableStateOf("13800001234") }
    var code by remember { mutableStateOf("") }
    var np by remember { mutableStateOf("") }
    var np2 by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    AuthCard("找回密码", "短信验证码重置 · 重置后需重新登录") {
        OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("手机号") },
            placeholder = { Text("请输入注册手机号") }, singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            modifier = Modifier.fillMaxWidth().padding(top = 4.dp))
        CodeField(code, countdown,
            onCode = { code = it },
            onSend = { sendResetCode(scope, phone) { err = it; countdown = 60 } })
        PwdField("新密码(≥10 位,含大小写与数字)", np, { np = it }, "请设置新密码")
        PwdField("确认新密码", np2, { np2 = it }, "请再次输入新密码")
        ResetFooter(err,
            onSubmit = { doReset(scope, nav, phone, code, np, np2) { err = it } },
            onBack = { nav.pop() })
    }
}

private fun sendResetCode(scope: kotlinx.coroutines.CoroutineScope, phone: String, onDone: (String) -> Unit) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    scope.launch {
        try {
            UserApi.auth.smsCode(phone, "reset"); onDone("验证码已发送")
        } catch (e: Exception) { onDone("验证码发送失败") }
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
            nav.replace(Route.Login) // 草稿: 重置成功跳回登录页
        } catch (e: Exception) { onErr("重置失败") }
    }
}

@Composable
private fun ResetFooter(err: String, onSubmit: () -> Unit, onBack: () -> Unit) {
    Text(err, fontSize = 12.5.sp, color = Palette.err, modifier = Modifier.padding(bottom = 6.dp, top = 2.dp))
    Button(
        onClick = onSubmit,
        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
        modifier = Modifier.fillMaxWidth().height(44.dp),
    ) { Text("确认重置") }
    Text("返回登录", fontSize = 13.sp, color = Palette.primary,
        modifier = Modifier.padding(top = 14.dp).clickable { onBack() })
}
