package com.ymm.boss.worker.ui

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
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material3.Icon
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.api.friendlyMessage
import com.ymm.boss.worker.push.PushRegistrar
import com.ymm.boss.worker.api.AuthApi
import com.ymm.boss.worker.api.ApiException
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import com.ymm.boss.worker.util.DevMode
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// 登录页(对齐用户端 designs/login-register-states-v2):
// 渐变 Hero + 悬浮白卡(仅分段+表单行),协议行/主按钮/入驻入口在卡外;
// 验证码/密码双模式,入驻入口降级为小字链接。

private val pageBg = Color(0xFFF6F8FA)

@Composable
fun LoginScreen(onLoggedIn: () -> Unit, onOnboard: () -> Unit, onAgreement: () -> Unit = {}) {
    var mode by remember { mutableStateOf("sms") }
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var pwdVisible by remember { mutableStateOf(false) }
    var agreed by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val dev = remember { DevMode.get(ctx) }
    var devMode by remember { mutableStateOf(dev.enabled) }

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    Column(
        Modifier.fillMaxSize().background(pageBg)
            .verticalScroll(rememberScrollState()),
    ) {
        Box(Modifier.fillMaxWidth()) {
            LoginHero()
            Column(
                Modifier.fillMaxWidth()
                    .padding(top = 176.dp)
                    .padding(horizontal = 16.dp),
            ) {
                // 悬浮白卡:分段+表单行
                Column(
                    Modifier.fillMaxWidth()
                        .background(Color.White, RoundedCornerShape(10.dp))
                        .padding(16.dp),
                ) {
                    AuthSegment(listOf("sms" to "验证码登录", "password" to "密码登录"), mode) {
                        mode = it; err = ""
                    }
                    AuthPhoneRow(phone, "请输入手机号") { phone = it }
                    if (mode == "sms") {
                        AuthCodeRow(code, countdown,
                            onCode = { code = it },
                            onSend = {
                                sendCode(scope, phone, onCode = { code = it }) { err = it; countdown = 59 }
                            })
                    } else {
                        AuthPwdRow(password, { password = it }, "请输入密码", pwdVisible) { pwdVisible = !pwdVisible }
                    }
                    if (err.isNotBlank()) AuthFootnote(err, Color(0xFFFF2D2F))
                }
                // 协议行
                AuthAgreeRow(agreed, onToggle = { agreed = it }, onAgreement = onAgreement)
                // 主按钮
                AuthPrimaryButton("登录", enabled = agreed && !busy, loading = busy,
                    onClick = {
                        busy = true
                        doLogin(scope, onLoggedIn, phone, mode, if (mode == "sms") code else password, devMode) {
                            err = it; busy = false
                        }
                    })
                // 入驻入口:小字链接,视觉层级最低(频率分层硬约束)
                OnboardEntry(onOnboard)
            }
        }
        Spacer(Modifier.height(32.dp))
    }
}

// 品牌渐变 Hero:160deg #0872F4→#0B82F8→#1698FA,圆角方标+名称+副文案+柔光圆
@Composable
private fun LoginHero() {
    Box(
        Modifier.fillMaxWidth().height(212.dp).background(
            Brush.linearGradient(listOf(Color(0xFF0872F4), Color(0xFF0B82F8), Color(0xFF1698FA)))),
    ) {
        // 柔和光效:两枚半透明白圆
        Box(Modifier.size(160.dp).align(Alignment.TopEnd).offset(x = 44.dp, y = (-44).dp)
            .background(Color.White.copy(alpha = 0.06f), CircleShape))
        Box(Modifier.size(110.dp).align(Alignment.CenterStart).offset(x = (-36).dp, y = 40.dp)
            .background(Color.White.copy(alpha = 0.08f), CircleShape))
        Column(
            Modifier.fillMaxWidth().padding(bottom = 48.dp).align(Alignment.Center),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier.size(64.dp)
                    .background(Color.White.copy(alpha = 0.18f), RoundedCornerShape(16.dp))
                    .border(1.dp, Color.White.copy(alpha = 0.35f), RoundedCornerShape(16.dp)),
                contentAlignment = Alignment.Center,
            ) { Icon(Icons.Outlined.Build, contentDescription = null, tint = Color.White, modifier = Modifier.size(32.dp)) }
            Text("装维全流程 · 师傅端", fontSize = 20.sp, fontWeight = FontWeight.Bold,
                color = Color.White, modifier = Modifier.padding(top = 14.dp))
            Text("工单接单 · 现场作业 · 扫码签收",
                style = androidx.compose.ui.text.TextStyle(
                    fontSize = 12.sp, letterSpacing = 1.sp,
                    color = Color.White.copy(alpha = 0.85f)),
                modifier = Modifier.padding(top = 6.dp))
        }
    }
}

// 入驻入口:小字链接,视觉层级最低
@Composable
private fun OnboardEntry(onOnboard: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 16.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text("还没有入驻？", fontSize = 12.sp, color = Muted)
        Text("申请入驻", fontSize = 12.sp, color = Primary, fontWeight = FontWeight.W500,
            modifier = Modifier.clickable { onOnboard() })
    }
}

private fun sendCode(
    scope: kotlinx.coroutines.CoroutineScope,
    phone: String,
    onCode: (String) -> Unit,
    onDone: (String) -> Unit,
) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    if (!Regex("^1\\d{10}$").matches(phone)) { onDone("手机号格式不正确"); return }
    scope.launch {
        try {
            AuthApi.smsCode(phone.trim())
            // 开发模式:自动回填
            val dev = com.ymm.boss.worker.util.DevMode.get(android.app.Activity())
            if (dev.enabled) {
                repeat(8) {
                    delay(300)
                    try {
                        val c = AuthApi.devSmsCode(phone.trim()).optString("code")
                        if (c.isNotEmpty()) { onCode(c); onDone(""); return@launch }
                    } catch (_: ApiException) { }
                }
            }
            onDone("")
        } catch (e: Exception) { onDone("验证码发送失败：${friendlyMessage(e)}") }
    }
}

private fun doLogin(
    scope: kotlinx.coroutines.CoroutineScope,
    onLoggedIn: () -> Unit,
    phone: String, mode: String, credential: String,
    devMode: Boolean,
    onErr: (String) -> Unit,
) {
    if (phone.isBlank() || credential.isBlank()) {
        onErr("请输入手机号与" + if (mode == "sms") "验证码" else "密码"); return
    }
    scope.launch {
        try {
            val r = AuthApi.login(phone.trim(), mode, credential.trim())
            val tk = r.optString("token")
            if (tk.isEmpty()) { onErr("登录响应缺少 token"); return@launch }
            Api.setToken(tk)
            PushRegistrar.ensureRegisteredOnLogin()
            onLoggedIn()
        } catch (e: Exception) {
            if (e is ApiException && e.status == 40100) {
                onErr(if (mode == "sms") "验证码错误或已过期，请重新获取" else "手机号或密码不正确")
            } else {
                onErr("登录失败：${friendlyMessage(e)}")
            }
        }
    }
}
