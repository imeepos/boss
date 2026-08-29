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
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Router
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.theme.RnPalette
import com.ymm.boss.user.util.devAutoFillSms
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** 登录页(designs/login-register-states-v2):渐变 Hero + 悬浮白卡(仅分段+表单行),
 *  协议行/主按钮/注册入口在卡外;验证码/密码双模式,注册降级为小字链接。 */
@Composable
fun LoginScreen(nav: Nav) {
    var mode by remember { mutableStateOf("sms") }
    var phone by remember { mutableStateOf("13800001234") }
    var code by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var pwdVisible by remember { mutableStateOf(false) }
    var agreed by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val context = androidx.compose.ui.platform.LocalContext.current

    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    Column(
        Modifier.fillMaxSize().background(RnPalette.pageBg)
            .verticalScroll(rememberScrollState()),
    ) {
        Box(Modifier.fillMaxWidth()) {
            LoginHero()
            Column(
                Modifier.fillMaxWidth()
                    .padding(top = 176.dp)
                    .padding(horizontal = 16.dp),
            ) {
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
                        Text("忘记密码", fontSize = 12.sp, color = RnPalette.muted,
                            modifier = Modifier.align(Alignment.End).padding(top = 6.dp)
                                .clickable { nav.push(com.ymm.boss.user.ui.Route.Forgot) })
                    }
                    if (err.isNotBlank()) RNFootnote(err, RnPalette.error)
                }
                AuthAgreeRow(agreed, onToggle = { agreed = it },
                    onAgreement = { nav.push(com.ymm.boss.user.ui.Route.Agreement) })
                RNPrimaryButton("登录", enabled = !busy, loading = busy,
                    onClick = {
                        if (!agreed) {
                            // OB-06:未勾协议给原因提示而非静默无响应(灰底禁用无解释,新用户困惑)
                            err = "请先阅读并同意《用户协议》"
                            return@RNPrimaryButton
                        }
                        busy = true
                        doLogin(scope, nav, context, phone, mode, if (mode == "sms") code else password) {
                            err = it; busy = false
                        }
                    })
                RegisterEntry { nav.push(com.ymm.boss.user.ui.Route.Register) }
            }
        }
        Spacer(Modifier.height(32.dp))
    }
}

/** 品牌渐变 Hero:160deg #0872F4→#0B82F8→#1698FA,圆角方标+名称+副文案+柔光圆。 */
@Composable
private fun LoginHero() {
    Box(
        Modifier.fillMaxWidth().height(212.dp).background(
            Brush.linearGradient(listOf(RnPalette.heroStart, RnPalette.heroMid, RnPalette.heroEnd))),
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
            ) { Icon(Icons.Outlined.Router, contentDescription = null, tint = Color.White, modifier = Modifier.size(32.dp)) }
            Text("装维全流程 · 用户端", fontSize = 20.sp, fontWeight = FontWeight.Bold,
                color = Color.White, modifier = Modifier.padding(top = 14.dp))
            Text("宽带办理 · 账单缴费 · 故障报修",
                style = androidx.compose.ui.text.TextStyle(
                    fontSize = 12.sp, letterSpacing = 1.sp,
                    color = Color.White.copy(alpha = 0.85f)),
                modifier = Modifier.padding(top = 6.dp))
        }
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
        Text("还没有账号？", fontSize = 12.sp, color = RnPalette.muted)
        Text("立即注册", fontSize = 12.sp, color = RnPalette.primary, fontWeight = FontWeight.W500,
            modifier = Modifier.clickable { onClick() })
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
            UserApi.auth.smsCode(phone, "login")
            devAutoFillSms(phone, "login", onCode)
            onDone("")
        } catch (e: Exception) { onDone("验证码发送失败：" + Api.friendlyMessage(e)) }
    }
}

private fun doLogin(
    scope: kotlinx.coroutines.CoroutineScope,
    nav: Nav, context: android.content.Context, phone: String, mode: String, credential: String,
    onErr: (String) -> Unit,
) {
    if (phone.isBlank() || credential.isBlank()) { onErr("请输入手机号与" + if (mode == "sms") "验证码" else "密码"); return }
    scope.launch {
        try {
            val tk = UserApi.auth.login(phone, mode, credential).optString("token")
            if (tk.isEmpty()) { onErr("登录响应缺少 token"); return@launch }
            Api.setToken(tk)
            // B 轨:上报设备注册(幂等,失败静默),独立协程不阻塞登录跳转。
            scope.launch { com.ymm.boss.user.api.PushApi.registerDevice(context) }
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
