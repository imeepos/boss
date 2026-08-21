package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
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
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** 登录页(designs/login-register-states-v2):渐变 Hero + 白卡表单,验证码/密码双模式,注册降级为小字链接。 */
@Composable
fun LoginScreen(nav: Nav) {
    var mode by remember { mutableStateOf("sms") }
    var phone by remember { mutableStateOf("13800001234") }
    var code by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var pwdVisible by remember { mutableStateOf(false) }
    var agreed by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    Column(
        Modifier.fillMaxSize().background(Color(0xFFF6F8FA))
            .verticalScroll(rememberScrollState()),
    ) {
        LoginHero()
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            Column(
                Modifier.fillMaxWidth()
                    .background(Color.White, RoundedCornerShape(10.dp))
                    .padding(16.dp),
            ) {
                LoginSegment(mode) { mode = it }
                LoginPhoneField(phone) { phone = it }
                if (mode == "sms") LoginCodeField(code, countdown,
                    onCode = { code = it },
                    onSend = { sendCode(scope, phone) { err = it; countdown = 59 } })
                else LoginPwdField(password, pwdVisible, { password = it }, { pwdVisible = !pwdVisible },
                    onForgot = { nav.push(com.ymm.boss.user.ui.Route.Forgot) })
                if (err.isNotBlank()) RNFootnote(err, Color(0xFFFF2D2F))
                LoginAgreeRow(agreed, onToggle = { agreed = it },
                    onAgreement = { nav.push(com.ymm.boss.user.ui.Route.Agreement) })
                RNPrimaryButton("登录", enabled = agreed,
                    onClick = { doLogin(scope, nav, phone, mode, if (mode == "sms") code else password) { err = it } })
            }
            RegisterEntry { nav.push(com.ymm.boss.user.ui.Route.Register) }
        }
        Spacer(Modifier.height(32.dp))
    }
}

/** 品牌渐变 Hero:160deg #0872F4→#0B82F8→#1698FA,品牌圆标+名称+副文案。 */
@Composable
private fun LoginHero() {
    Box(
        Modifier.fillMaxWidth().height(210.dp).background(
            Brush.linearGradient(listOf(Color(0xFF0872F4), Color(0xFF0B82F8), Color(0xFF1698FA)))),
        contentAlignment = Alignment.Center,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                Modifier.size(56.dp).background(Color.White.copy(alpha = 0.2f), CircleShape),
                contentAlignment = Alignment.Center,
            ) { Text("Boss", color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.Bold) }
            Text("装维全流程 · 用户端", fontSize = 20.sp, fontWeight = FontWeight.Bold,
                color = Color.White, modifier = Modifier.padding(top = 12.dp))
            Text("宽带办理 · 账单缴费 · 故障报修", fontSize = 12.sp,
                color = Color.White.copy(alpha = 0.85f), modifier = Modifier.padding(top = 6.dp))
        }
    }
}

/** iOS 风格分段控件:灰底胶囊容器,选中白底蓝字。 */
@Composable
private fun LoginSegment(mode: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(bottom = 4.dp)
            .background(RN.line, RoundedCornerShape(999.dp))
            .padding(3.dp),
    ) {
        listOf("sms" to "验证码登录", "password" to "密码登录").forEach { (k, label) ->
            val active = mode == k
            Text(
                label, fontSize = 13.sp,
                color = if (active) RN.primary else RN.muted,
                fontWeight = if (active) FontWeight.W600 else FontWeight.Normal,
                modifier = Modifier.weight(1f)
                    .background(if (active) Color.White else Color.Transparent, RoundedCornerShape(999.dp))
                    .clickable { onSelect(k) }
                    .padding(vertical = 8.dp),
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            )
        }
    }
}

@Composable
private fun LoginPhoneField(phone: String, onChange: (String) -> Unit) {
    OutlinedTextField(
        value = phone, onValueChange = onChange, label = { Text("手机号") },
        placeholder = { Text("请输入手机号") }, singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
        modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
    )
}

/** 验证码行:右"获取验证码/59s后重发"。 */
@Composable
private fun LoginCodeField(code: String, countdown: Int, onCode: (String) -> Unit, onSend: () -> Unit) {
    OutlinedTextField(
        value = code, onValueChange = onCode, label = { Text("验证码") },
        placeholder = { Text("6 位验证码") }, singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
        modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
        trailingIcon = {
            Text(
                if (countdown > 0) "${countdown}s后重发" else "获取验证码",
                fontSize = 12.sp, fontWeight = FontWeight.W500,
                color = if (countdown > 0) RN.placeholder else RN.primary,
                modifier = if (countdown > 0) Modifier.padding(end = 8.dp)
                else Modifier.padding(end = 8.dp).clickable { onSend() },
            )
        },
    )
}

/** 密码行:眼睛图标切换明文;行下"忘记密码"链接(spec: 密码态专属)。 */
@Composable
private fun LoginPwdField(
    pwd: String, visible: Boolean, onChange: (String) -> Unit, onToggle: () -> Unit,
    onForgot: () -> Unit,
) {
    Column {
        OutlinedTextField(
            value = pwd, onValueChange = onChange, label = { Text("密码") },
            singleLine = true,
            visualTransformation = if (visible) VisualTransformation.None else PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
            trailingIcon = {
                Icon(
                    if (visible) Icons.Filled.Visibility else Icons.Filled.VisibilityOff,
                    contentDescription = null, tint = RN.placeholder,
                    modifier = Modifier.padding(end = 8.dp).clickable { onToggle() })
            },
            modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
        )
        Text("忘记密码", fontSize = 12.sp, color = RN.primary, fontWeight = FontWeight.W500,
            modifier = Modifier.align(Alignment.End).padding(top = 2.dp).clickable { onForgot() })
    }
}

/** 协议勾选:圈选样式,未勾选禁用登录;协议名跳转协议页。 */
@Composable
private fun LoginAgreeRow(agreed: Boolean, onToggle: (Boolean) -> Unit, onAgreement: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(top = 10.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.size(18.dp).clickable { onToggle(!agreed) }
                .border(1.dp, if (agreed) RN.primary else RN.placeholder, CircleShape)
                .padding(4.dp),
            contentAlignment = Alignment.Center,
        ) { if (agreed) Box(Modifier.size(8.dp).background(RN.primary, CircleShape)) }
        Spacer(Modifier.width(8.dp))
        Text("已阅读并同意", fontSize = 12.sp, color = RN.muted)
        Text("《用户协议》", fontSize = 12.sp, color = RN.primary,
            fontWeight = FontWeight.W500, modifier = Modifier.clickable { onAgreement() })
    }
}

/** 注册入口:小字链接,视觉层级最低(频率分层硬约束)。 */
@Composable
private fun RegisterEntry(onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 16.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text("还没有账号？", fontSize = 12.sp, color = RN.muted)
        Text("立即注册", fontSize = 12.sp, color = RN.primary, fontWeight = FontWeight.W500,
            modifier = Modifier.clickable { onClick() })
    }
}

private fun sendCode(scope: kotlinx.coroutines.CoroutineScope, phone: String, onDone: (String) -> Unit) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    if (!Regex("^1\\d{10}$").matches(phone)) { onDone("手机号格式不正确"); return }
    scope.launch {
        try {
            UserApi.auth.smsCode(phone, "login"); onDone("")
        } catch (e: Exception) { onDone("验证码发送失败：" + Api.friendlyMessage(e)) }
    }
}

private fun doLogin(
    scope: kotlinx.coroutines.CoroutineScope,
    nav: Nav, phone: String, mode: String, credential: String,
    onErr: (String) -> Unit,
) {
    if (phone.isBlank() || credential.isBlank()) { onErr("请输入手机号与" + if (mode == "sms") "验证码" else "密码"); return }
    scope.launch {
        try {
            val tk = UserApi.auth.login(phone, mode, credential).optString("token")
            if (tk.isEmpty()) { onErr("登录响应缺少 token"); return@launch }
            Api.setToken(tk)
            nav.resetTo(com.ymm.boss.user.ui.Route.Home)
        } catch (e: Exception) { onErr(loginErrorMessage(e, mode)) }
    }
}

/** 登录场景错误细化:40100 按模式区分(验证码 vs 密码),其余按通用映射。 */
internal fun loginErrorMessage(e: Exception, mode: String): String {
    if (e is Api.HttpError && e.status == 40100) {
        return if (mode == "sms") "验证码错误或已过期，请重新获取" else "手机号或密码不正确"
    }
    return Api.friendlyMessage(e)
}
