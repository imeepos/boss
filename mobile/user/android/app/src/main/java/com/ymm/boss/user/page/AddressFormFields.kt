package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Palette

// 地址编辑弹窗的表单原子组件;从 AddressEditorSheet 拆出守 300 行红线(2026-08-26)。

@Composable
internal fun SheetHeader(title: String, onClose: () -> Unit) {
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(title, fontSize = 16.sp, fontWeight = FontWeight.W600, color = Palette.ink)
        IconButton(onClick = onClose) {
            Icon(Icons.Outlined.Close, contentDescription = "关闭", tint = Palette.muted,
                modifier = Modifier.size(20.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun CommunityField(value: String, options: List<String>, onChange: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(
        expanded = expanded,
        onExpandedChange = { expanded = !expanded },
        modifier = Modifier.fillMaxWidth(),
    ) {
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            placeholder = { Text("请输入小区名", fontSize = 13.sp, color = Palette.subtle) },
            singleLine = true,
            shape = RoundedCornerShape(10.dp),
            trailingIcon = {
                if (value.isNotBlank()) {
                    IconButton(onClick = { onChange(""); expanded = false }) {
                        Icon(Icons.Outlined.Close, contentDescription = "清除",
                            tint = Palette.muted, modifier = Modifier.size(18.dp))
                    }
                } else {
                    ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded)
                }
            },
            modifier = Modifier.fillMaxWidth().menuAnchor(MenuAnchorType.PrimaryNotEditable)
                .padding(bottom = 8.dp),
        )
        // 必须在 ExposedDropdownMenuBoxScope 内调用 ExposedDropdownMenu，否则
        // DropdownMenuItem 会渲染到 anchor 里（真机已重现 placeholder 跟"暂无历史小区"重叠）。
        ExposedDropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false },
        ) {
            if (options.isEmpty()) {
                DropdownMenuItem(
                    text = { Text("暂无历史小区", fontSize = 13.sp, color = Palette.muted) },
                    onClick = { expanded = false },
                    enabled = false,
                )
            } else {
                options.forEach { item ->
                    DropdownMenuItem(
                        text = { Text(item, fontSize = 13.sp, color = Palette.ink) },
                        onClick = { onChange(item); expanded = false },
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun AddrInput(value: String, placeholder: String, keyboard: KeyboardType, onChange: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        placeholder = { Text(placeholder, fontSize = 13.sp, color = Palette.subtle) },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = keyboard),
        shape = RoundedCornerShape(10.dp),
        modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
    )
}

@Composable
internal fun SubmitButton(text: String, enabled: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = enabled,
        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary,
            disabledContainerColor = Palette.primary.copy(alpha = 0.4f)),
        shape = RoundedCornerShape(10.dp),
        modifier = Modifier.fillMaxWidth().height(44.dp),
    ) {
        Text(text, fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.panel)
    }
}

// 菲律宾市场手机号:09 开头共 11 位(如 09171234567);留空=回退账户手机号,语义保留。
internal fun phoneOk(phone: String): Boolean {
    if (phone.isBlank()) return true
    return Regex("^09\\d{9}$").matches(phone.trim())
}

@Composable
internal fun FieldLabel(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.muted, modifier = Modifier.padding(bottom = 4.dp))
}
