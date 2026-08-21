package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.PhoneAndroid
import androidx.compose.material.icons.outlined.Shield
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary

private val fieldBg = Color(0xFFF7F8FA)
private val iconTint = Color(0xFF7D8593)

// 登录/注册共用认证表单组件(对齐用户端 AuthForm):
// 浅灰填充圆角输入行(无下划线/无浮动标签,focus 蓝描边) + 灰槽蓝块分段控件 + 圆形协议勾选。

// 单行填充输入:52dp 高,#F7F8FA 底,#E7EAF0 描边,12dp 圆角,左线性图标,内置占位文字。
@Composable
internal fun AuthInputRow(
    value: String, onChange: (String) -> Unit, hint: String,
    icon: ImageVector, keyboardType: KeyboardType,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    trailing: (@Composable () -> Unit)? = null,
) {
    var focused by remember { mutableStateOf(false) }
    val shape: Shape = RoundedCornerShape(12.dp)
    Row(
        Modifier.fillMaxWidth().padding(top = 8.dp).height(52.dp)
            .background(fieldBg, shape)
            .border(1.dp, if (focused) Primary else Line, shape)
            .padding(horizontal = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = iconTint, modifier = Modifier.size(20.dp))
        BasicTextField(
            value = value, onValueChange = onChange, singleLine = true,
            textStyle = TextStyle(fontSize = 15.sp, color = Ink),
            keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
            visualTransformation = visualTransformation,
            modifier = Modifier.weight(1f).padding(horizontal = 10.dp)
                .onFocusChanged { focused = it.isFocused },
            decorationBox = { inner ->
                Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
                    if (value.isEmpty()) Text(hint, fontSize = 15.sp, color = Muted)
                    inner()
                }
            },
        )
        trailing?.invoke()
    }
}

// 手机号行
@Composable
internal fun AuthPhoneRow(value: String, hint: String, onChange: (String) -> Unit) {
    AuthInputRow(value, onChange, hint, Icons.Outlined.PhoneAndroid, KeyboardType.Phone)
}

// 验证码行:右侧"获取验证码/倒计时"文字按钮
@Composable
internal fun AuthCodeRow(code: String, countdown: Int, onCode: (String) -> Unit, onSend: () -> Unit) {
    AuthInputRow(code, onCode, "请输入验证码", Icons.Outlined.Shield, KeyboardType.NumberPassword,
        trailing = {
            Text(
                if (countdown > 0) "${countdown}s后重发" else "获取验证码",
                fontSize = 13.sp, fontWeight = FontWeight.W500,
                color = if (countdown > 0) Muted else Primary,
                modifier = Modifier.clickable(enabled = countdown == 0) { onSend() },
            )
        })
}

// 密码行:右侧眼睛切换明文
@Composable
internal fun AuthPwdRow(
    value: String, onChange: (String) -> Unit, hint: String,
    visible: Boolean, onToggle: () -> Unit,
) {
    AuthInputRow(value, onChange, hint, Icons.Outlined.Lock, KeyboardType.Password,
        visualTransformation = if (visible) VisualTransformation.None else PasswordVisualTransformation(),
        trailing = {
            Icon(
                if (visible) Icons.Filled.Visibility else Icons.Filled.VisibilityOff,
                contentDescription = null, tint = Muted,
                modifier = Modifier.size(20.dp).clickable { onToggle() },
            )
        })
}

// 分段控件:灰底浅圆角槽,选中蓝实底白字(对齐设计稿,非胶囊)
@Composable
internal fun AuthSegment(options: List<Pair<String, String>>, selected: String, onSelect: (String) -> Unit) {
    Row(
        Modifier.fillMaxWidth()
            .background(Color(0xFFF5F6F8), RoundedCornerShape(10.dp))
            .padding(3.dp),
    ) {
        options.forEach { (k, label) ->
            val active = selected == k
            Text(
                label, fontSize = 14.sp, textAlign = TextAlign.Center,
                color = if (active) Color.White else Muted,
                fontWeight = if (active) FontWeight.W600 else FontWeight.Normal,
                modifier = Modifier.weight(1f)
                    .background(if (active) Primary else Color.Transparent, RoundedCornerShape(8.dp))
                    .clickable { onSelect(k) }
                    .padding(vertical = 9.dp),
            )
        }
    }
}

// 协议勾选行:圆形勾选框 + 居中"已阅读并同意《用户协议》"
@Composable
internal fun AuthAgreeRow(
    agreed: Boolean, onToggle: (Boolean) -> Unit, onAgreement: () -> Unit,
) {
    Row(
        Modifier.fillMaxWidth().padding(top = 16.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier.size(18.dp).clickable { onToggle(!agreed) }
                .border(1.dp, if (agreed) Primary else Color(0xFFD1D7E0), CircleShape)
                .padding(4.dp),
            contentAlignment = Alignment.Center,
        ) { if (agreed) Box(Modifier.size(8.dp).background(Primary, CircleShape)) }
        Spacer(Modifier.width(8.dp))
        Text("我已阅读并同意", fontSize = 12.sp, color = Muted)
        Text("《服务协议》", fontSize = 12.sp, color = Primary,
            fontWeight = FontWeight.W500, modifier = Modifier.clickable { onAgreement() })
    }
}

// 主操作按钮:≥44dp,禁用灰,loading 转圈
@Composable
internal fun AuthPrimaryButton(text: String, enabled: Boolean, loading: Boolean = false, onClick: () -> Unit) {
    val bg = if (enabled) Primary else Muted
    Box(
        Modifier.fillMaxWidth().padding(vertical = 4.dp).height(46.dp)
            .background(bg, RoundedCornerShape(10.dp))
            .clickable(enabled = enabled && !loading) { onClick() },
        contentAlignment = Alignment.Center,
    ) {
        if (loading) CircularProgressIndicator(
            Modifier.size(20.dp), color = Color.White, strokeWidth = 2.dp)
        else Text(text, fontSize = 15.sp, color = Color.White, fontWeight = FontWeight.W600)
    }
}

// 辅助/说明小字
@Composable
internal fun AuthFootnote(text: String, color: Color = Muted) {
    Text(text, fontSize = 12.sp, color = color, modifier = Modifier.padding(vertical = 4.dp))
}
