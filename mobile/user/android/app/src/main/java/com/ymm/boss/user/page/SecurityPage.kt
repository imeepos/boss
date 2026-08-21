package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import com.ymm.boss.user.util.devAutoFillSms
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/security.html:账号安全,GET /profile/security + 改密码/换绑手机表单。
@Composable
fun SecurityScreen(nav: Nav) {
    var data by remember { mutableStateOf<JSONObject?>(null) }
    LaunchedEffect(nav.refreshTick) {
        try { data = ProfileApi.security() } catch (e: Exception) { data = null }
    }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("账号安全", onBack = { nav.pop() })
        RealNameCard(data, nav)
        PasswordCard(data)
        PhoneCard(data)
        PolicyCard()
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun RealNameCard(data: JSONObject?, nav: Nav) {
    val verified = data?.optString("realNameStatus") == "VERIFIED"
    AppCard {
        CardTitle("实名认证", "核验详情") { nav.push(Route.Verify) }
        Row(Modifier.padding(top = 4.dp)) { Tag(if (verified) "已实名" else "待补登", if (verified) Palette.success else Palette.muted) }
        CellRow("姓名", right = { GrayText(data?.optString("nameMasked") ?: "—") })
        CellRow("证件号码", right = { GrayText(data?.optString("idNoMasked") ?: "—") })
        CellRow("认证时间", right = { GrayText(data?.optString("verifyAt") ?: "—") })
        Notice("实名信息用于办理入网与停机复机，一经核验不可在线变更，如需变更请携证件至营业厅。")
    }
}

@Composable
private fun GrayText(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.muted)
}

@Composable
private fun PasswordCard(data: JSONObject?) {
    var open by remember { mutableStateOf(false) }
    AppCard {
        CardTitle("登录密码")
        CellRow("登录密码", desc = "上次修改 ${data?.optString("passwordUpdatedAt").orEmpty().ifBlank { "—" }}",
            right = { SmallButton(if (open) "收起" else "修改") { open = !open } })
        if (open) PasswordForm()
    }
}

@Composable
private fun PasswordForm() {
    val scope = rememberCoroutineScope()
    var oldPwd by remember { mutableStateOf("") }
    var newPwd by remember { mutableStateOf("") }
    var msg by remember { mutableStateOf("") }
    Column(Modifier.padding(top = 8.dp)) {
        OutlinedTextField(value = oldPwd, onValueChange = { oldPwd = it }, label = { Text("当前密码") },
            singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = newPwd, onValueChange = { newPwd = it }, label = { Text("新密码(≥10位,含大小写与数字)") },
            singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
        Notice(msg)
        Button(
            onClick = {
                scope.launch {
                    try { ProfileApi.changePassword(oldPwd, newPwd); msg = "密码已修改" }
                    catch (e: Exception) { msg = "修改失败,请检查当前密码" }
                }
            },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().height(42.dp),
        ) { Text("确认修改") }
    }
}

@Composable
private fun PhoneCard(data: JSONObject?) {
    var open by remember { mutableStateOf(false) }
    AppCard {
        CardTitle("绑定手机号")
        CellRow("绑定手机号", desc = data?.optString("phoneMasked").orEmpty().ifBlank { "—" },
            right = { SmallButton(if (open) "收起" else "更换") { open = !open } })
        if (open) PhoneForm()
    }
}

@Composable
private fun PhoneForm() {
    val scope = rememberCoroutineScope()
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var msg by remember { mutableStateOf("") }
    Column(Modifier.padding(top = 8.dp)) {
        OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("新手机号") },
            singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone), modifier = Modifier.fillMaxWidth())
        Row(Modifier.fillMaxWidth().padding(top = 8.dp)) {
            OutlinedTextField(value = code, onValueChange = { code = it }, label = { Text("短信验证码") },
                singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
                modifier = Modifier.weight(1f))
            OutlinedButton(
                onClick = {
                    scope.launch {
                        try {
                            UserApi.auth.smsCode(phone, "login")
                            devAutoFillSms(phone, "login") { code = it }
                            msg = "验证码已发送"
                        }
                        catch (e: Exception) { msg = "验证码发送失败" }
                    }
                },
            ) { Text("获取验证码", fontSize = 12.5.sp, color = Palette.primary) }
        }
        Notice(msg)
        Button(
            onClick = {
                scope.launch {
                    try { ProfileApi.changePhone(phone, code); msg = "手机号已更换" }
                    catch (e: Exception) { msg = "更换失败,请稍后重试" }
                }
            },
            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
            modifier = Modifier.fillMaxWidth().height(42.dp),
        ) { Text("确认更换") }
    }
}

@Composable
private fun PolicyCard() {
    AppCard {
        CardTitle("安全提示")
        CellRow("密码策略", desc = "≥10 位，含大小写与数字，90 天更换")
        CellRow("登录保护", desc = "连续失败 5 次锁定 30 分钟")
    }
}

@Composable
private fun SmallButton(label: String, onClick: () -> Unit) {
    Button(onClick = onClick, colors = ButtonDefaults.buttonColors(containerColor = Palette.primary)) {
        Text(label, fontSize = 12.5.sp)
    }
}
