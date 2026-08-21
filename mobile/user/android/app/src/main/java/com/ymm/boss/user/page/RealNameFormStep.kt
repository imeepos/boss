package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.VerifiedUser
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.api.Api
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.json.JSONObject

// Screen1 填写信息:安全说明卡 + 表单卡(姓名/证件号/绑定手机号只读/验证码) + 协议勾选 + 下一步。
@Composable
internal fun RNFormStep(
    data: JSONObject?, name: String, idNo: String, sms: String,
    onName: (String) -> Unit, onIdNo: (String) -> Unit, onSms: (String) -> Unit,
    onNext: () -> Unit, onAgreement: () -> Unit,
) {
    var err by remember { mutableStateOf("") }
    var notice by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(false) }
    var countdown by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    LaunchedEffect(countdown) { while (countdown > 0) { delay(1000); countdown-- } }

    val phoneMasked = data?.optString("phoneMasked").orEmpty().ifBlank { "—" }
    SafetyCard()
    Column(Modifier.fillMaxWidth().padding(top = 8.dp)) {
        CardHead("认证信息")
        RNField("真实姓名", name, { onName(it) }, "请输入与证件一致的真实姓名",
            KeyboardType.Text, err == "name")
        RNField("身份证号", idNo, { onIdNo(it) }, "请输入 18 位身份证号码",
            KeyboardType.Number, err == "idNo")
        ReadOnlyField("手机号（绑定手机，不可修改）", phoneMasked)
        SmsField(sms, countdown, onSms = {
            onSms(it); if (err == "sms") err = ""
        }, onSend = {
            notice = ""
            sendVerifyCode(scope) { msg, cd ->
                if (cd) countdown = 59 else notice = msg  // spec: 起始即 "59s后重发"
            }
        })
        if (notice.isNotBlank()) RNFootnote(notice, Color(0xFFFF2D2F))
        if (err in setOf("name", "idNo", "sms")) RNFootnote(when (err) {
            "name" -> "请填写真实姓名"; "idNo" -> "身份证号需 18 位"; else -> "请输入短信验证码"
        }, Color(0xFFFF2D2F))
    }
    AgreeRow(agreed, onToggle = { agreed = it; if (err == "agree") err = "" },
        onAgreement = onAgreement)
    if (err == "agree") RNFootnote("请先阅读并同意认证服务协议", Color(0xFFFF2D2F))
    RNPrimaryButton("下一步", enabled = agreed) {
        val e = when {
            name.isBlank() -> "name"
            idNo.length != 18 -> "idNo"
            sms.isBlank() -> "sms"
            !agreed -> "agree"
            else -> ""
        }
        if (e.isEmpty()) onNext() else err = e
    }
}

private fun sendVerifyCode(scope: kotlinx.coroutines.CoroutineScope, onDone: (String, Boolean) -> Unit) {
    scope.launch {
        try {
            AccountApi.sendVerifySmsCode(); onDone("", true)
        } catch (e: Exception) { onDone(Api.friendlyMessage(e) + "，稍后可重试", false) }
    }
}

@Composable
private fun SafetyCard() {
    RNSharedCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Filled.VerifiedUser, contentDescription = null,
                tint = RN.primary, modifier = Modifier.size(22.dp))
            Spacer(Modifier.size(8.dp))
            Column {
                Text("信息安全保障", fontSize = 14.sp, fontWeight = FontWeight.W600, color = RN.ink)
                RNFootnote("认证信息加密传输，仅用于入网实名登记，不会用于其他用途")
            }
        }
    }
}

@Composable
internal fun CardHead(title: String) {
    Text(title, fontSize = 16.sp, fontWeight = FontWeight.Bold, color = RN.ink,
        modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp))
}

@Composable
private fun RNField(
    label: String, value: String, onChange: (String) -> Unit, placeholder: String,
    type: KeyboardType, isError: Boolean,
) {
    OutlinedTextField(
        value = value, onValueChange = onChange, label = { Text(label) },
        placeholder = { Text(placeholder) }, singleLine = true, isError = isError,
        keyboardOptions = KeyboardOptions(keyboardType = type),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
    )
}

@Composable
private fun ReadOnlyField(label: String, value: String) {
    OutlinedTextField(
        value = value, onValueChange = {}, readOnly = true, label = { Text(label) },
        singleLine = true,
        textStyle = TextStyle(fontSize = 15.sp, fontWeight = FontWeight.Medium, color = RN.muted),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
    )
}

/** 验证码行:右侧"获取验证码/59s后重发"文字按钮(default/倒计时/禁用)。 */
@Composable
private fun SmsField(code: String, countdown: Int, onSms: (String) -> Unit, onSend: () -> Unit) {
    OutlinedTextField(
        value = code, onValueChange = onSms, label = { Text("短信验证码") },
        placeholder = { Text("请输入验证码") }, singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
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

@Composable
private fun AgreeRow(agreed: Boolean, onToggle: (Boolean) -> Unit, onAgreement: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(top = 10.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.size(18.dp).clickable { onToggle(!agreed) }
                .border(1.dp, if (agreed) RN.primary else RN.placeholder, CircleShape)
                .padding(4.dp),
            contentAlignment = Alignment.Center,
        ) {
            if (agreed) Box(Modifier.size(8.dp).background(RN.primary, CircleShape))
        }
        Spacer(Modifier.size(8.dp))
        Text("已阅读并同意", fontSize = 12.sp, color = RN.muted)
        Text("《实名认证服务协议》", fontSize = 12.sp, color = RN.primary,
            fontWeight = FontWeight.W500, modifier = Modifier.clickable { onAgreement() })
        Text("，认证信息真实有效", fontSize = 12.sp, color = RN.muted)
    }
}
