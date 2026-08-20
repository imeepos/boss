package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.api.AuthApi
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import android.util.Log
private const val TAG = "WorkerLogin"

// 对齐 docs/worker/login.html:手机号 + 短信验证码登录
@Composable
fun LoginScreen(onLoggedIn: () -> Unit) {
    var phone by remember { mutableStateOf("") }
    var sms by remember { mutableStateOf("") }
    var tip by remember { mutableStateOf("") }
    var countdown by remember { mutableStateOf(0) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        Header("装维平台 · 师傅端", "请用工号手机号登录")
        Card(Modifier.padding(14.dp)) {
            FieldLabel("手机号")
            OutlinedTextField(value = phone, onValueChange = { phone = it },
                placeholder = { Text("请输入手机号") }, singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(12.dp))
            FieldLabel("验证码")
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(value = sms, onValueChange = { sms = it },
                    placeholder = { Text("短信验证码") }, singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.weight(1f))
                Text(if (countdown > 0) "${countdown}s 后重发" else "获取验证码",
                    fontSize = 13.sp, color = Primary,
                    modifier = Modifier
                        .background(Color(0xFFF6FFED), RoundedCornerShape(8.dp))
                        .clickable(countdown <= 0) {
                            if (phone.isBlank()) { tip = "请先输入手机号"; return@clickable }
                            scope.launch {
                                try { AuthApi.smsCode(phone.trim()) } catch (e: Exception) { tip = "验证码发送失败:${e.message}" }
                                countdown = 60
                                while (countdown > 0) { delay(1000); countdown-- }
                            }
                        }
                        .padding(horizontal = 12.dp, vertical = 12.dp))
            }
            Spacer(Modifier.height(16.dp))
            PrimaryButton("登录", enabled = !busy, modifier = Modifier.fillMaxWidth()) {
                if (phone.isBlank() || sms.isBlank()) { tip = "请输入手机号与验证码"; return@PrimaryButton }
                busy = true
                Log.d(TAG, "login start")
                scope.launch {
                    try {
                        val r = AuthApi.login(phone.trim(), "sms", sms.trim())
                        // Api 已解信封,r 即 data:{token,workerId}(契约 auth.yaml)
                        val tk = r.optString("token")
                        if (tk.isEmpty()) {
                            tip = "登录响应缺少 token"
                            busy = false
                            return@launch
                        }
                        Api.setToken(tk)
                        onLoggedIn()
                    } catch (e: Exception) {
                        tip = "登录失败:${e.message}"
                        busy = false
                    }
                }
            }
        }
        Card(Modifier.padding(14.dp)) {
            Notice("登录需师傅手机号在职且验证码有效;登录后 token 由 Api 自动携带。")
        }
        if (tip.isNotEmpty()) Card(Modifier.padding(14.dp)) { Notice(tip, red = true) }
    }
}

@Composable
private fun Header(title: String, sub: String) {
    Column(Modifier.fillMaxWidth()
        .background(Brush.linearGradient(listOf(Primary, Primary2)))
        .padding(horizontal = 16.dp, vertical = 24.dp)) {
        Text(title, color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(14.dp))
        StatusLine(sub)
    }
}

@Composable
fun FieldLabel(text: String) {
    Text(text, fontSize = 13.sp, color = Color(0xFF8C8C8C), modifier = Modifier.padding(bottom = 6.dp))
}
