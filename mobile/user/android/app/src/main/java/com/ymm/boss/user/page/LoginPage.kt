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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PillTab
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** 对应草稿 docs/user/login.html:验证码/密码双模式登录。 */
@Composable
fun LoginScreen(nav: Nav) {
    var mode by remember { mutableStateOf("sms") }
    var phone by remember { mutableStateOf("13800001234") }
    var code by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(true) }
    var err by remember { mutableStateOf("") }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(countdown) {
        while (countdown > 0) { delay(1000); countdown-- }
    }

    Box(
        Modifier.fillMaxSize().background(Palette.bg).verticalScroll(rememberScrollState()).padding(horizontal = 16.dp),
        contentAlignment = Alignment.TopCenter,
    ) {
        Column(
            Modifier.background(Palette.panel, RoundedCornerShape(12.dp)).padding(16.dp).fillMaxWidth(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(24.dp))
            Box(
                Modifier.size(56.dp).background(Palette.primary, CircleShape),
                contentAlignment = Alignment.Center,
            ) { Text("Boss", color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.Bold) }
            Text("装维全流程 · 用户端", fontSize = 20.sp, fontWeight = FontWeight.Bold, color = Palette.ink,
                modifier = Modifier.padding(top = 12.dp))
            Text("宽带办理 · 账单缴费 · 故障报修", fontSize = 12.5.sp, color = Palette.muted,
                modifier = Modifier.padding(top = 6.dp, bottom = 20.dp))

            OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("手机号") },
                singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                modifier = Modifier.fillMaxWidth())

            ModeSegment(mode) { mode = it }

            if (mode == "sms") {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(value = code, onValueChange = { code = it }, label = { Text("验证码") },
                        singleLine = true, modifier = Modifier.weight(1f),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword))
                    OutlinedButton(
                        onClick = { sendCode(scope, phone) { err = it; countdown = 60 } },
                        enabled = countdown == 0,
                    ) { Text(if (countdown > 0) "${countdown}s" else "获取验证码", fontSize = 12.5.sp, color = Palette.primary) }
                }
            } else {
                OutlinedTextField(value = password, onValueChange = { password = it }, label = { Text("密码") },
                    singleLine = true, visualTransformation = PasswordVisualTransformation(),
                    modifier = Modifier.fillMaxWidth())
            }

            Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                    Checkbox(checked = agreed, onCheckedChange = { agreed = it })
                    Text("已阅读并同意《用户协议》", fontSize = 12.5.sp, color = Palette.muted)
                }
                Text("忘记密码", fontSize = 12.5.sp, color = Palette.primary,
                    modifier = Modifier.clickable { nav.push(com.ymm.boss.user.ui.Route.Forgot) })
            }

            Text(err, fontSize = 12.5.sp, color = Palette.err, modifier = Modifier.padding(bottom = 6.dp))

            Button(
                onClick = { doLogin(scope, nav, phone, mode, if (mode == "sms") code else password, agreed) { err = it } },
                colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                modifier = Modifier.fillMaxWidth().height(44.dp),
            ) { Text("登录") }

            Spacer(Modifier.height(18.dp))
            DividerText("还没有账号")
            OutlinedButton(onClick = { nav.push(com.ymm.boss.user.ui.Route.Register) }, modifier = Modifier.fillMaxWidth().height(44.dp)) { Text("自助注册", color = Palette.primary) }
            Spacer(Modifier.height(14.dp))
            DividerText("其他登录方式")
            OutlinedButton(onClick = {}, modifier = Modifier.fillMaxWidth().height(44.dp)) { Text("本机号码一键登录", color = Palette.primary) }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
private fun ModeSegment(mode: String, onSelect: (String) -> Unit) {
    // 与账单/消息中心筛选同款 PillTab plain 形态,不再自绘文字 tab
    Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        listOf("sms" to "验证码登录", "password" to "密码登录").forEach { (k, label) ->
            PillTab(label, active = mode == k, onClick = { onSelect(k) }, plain = true)
        }
    }
}

@Composable
private fun DividerText(text: String) {
    Text(text, fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(bottom = 8.dp))
}

private fun sendCode(scope: kotlinx.coroutines.CoroutineScope, phone: String, onDone: (String) -> Unit) {
    if (phone.isBlank()) { onDone("请输入手机号"); return }
    if (!Regex("^1\\d{10}$").matches(phone)) { onDone("手机号格式不正确"); return }
    scope.launch {
        try {
            UserApi.auth.smsCode(phone, "login")
            onDone("验证码已发送")
        } catch (e: Exception) { onDone(Api.friendlyMessage(e).let { "验证码发送失败：$it" }) }
    }
}

private fun doLogin(
    scope: kotlinx.coroutines.CoroutineScope,
    nav: Nav, phone: String, mode: String, credential: String,
    agreed: Boolean, onErr: (String) -> Unit,
) {
    if (!agreed) { onErr("请先同意用户协议"); return }
    if (phone.isBlank() || credential.isBlank()) { onErr("请输入手机号与" + if (mode == "sms") "验证码" else "密码"); return }
    scope.launch {
        try {
            val r = UserApi.auth.login(phone, mode, credential)
            // Api.request 已解 {code,msg,data} 信封,这里直接取字段
            val tk = r.optString("token")
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
