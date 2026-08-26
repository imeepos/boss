package com.ymm.boss.user.page

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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
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
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Palette
import org.json.JSONObject

/**
 * 新增/编辑地址底部弹窗:小区、楼栋、门牌、联系人、手机号(label 由后端拼装)。
 * 提交前基础校验:小区/联系人非空;手机号格式正确。
 * existing==null → 新增;否则编辑。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddressEditorSheet(
    initial: JSONObject?,
    onDismiss: () -> Unit,
    onSubmit: (JSONObject) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var community by remember { mutableStateOf(initial?.optString("community").orEmpty()) }
    var building by remember { mutableStateOf(initial?.optString("building").orEmpty()) }
    var door by remember { mutableStateOf(initial?.optString("door").orEmpty()) }
    var contact by remember { mutableStateOf(initial?.optString("contact").orEmpty()) }
    var phone by remember { mutableStateOf("") } // phoneMasked 不回填,避免误导;后端空值取账户默认手机
    var err by remember { mutableStateOf("") }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Palette.panel,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
            SheetHeader(title = if (initial == null) "新增家庭地址" else "编辑家庭地址", onClose = onDismiss)
            Spacer(Modifier.height(8.dp))
            FieldLabel("小区 / 楼盘")
            AddrInput(community, "请输入小区名", KeyboardType.Text) { community = it; err = "" }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.weight(1f)) {
                    FieldLabel("楼栋")
                    AddrInput(building, "楼栋号", KeyboardType.Text) { building = it }
                }
                Box(Modifier.weight(1.4f)) {
                    FieldLabel("门牌号")
                    AddrInput(door, "单元/房间号", KeyboardType.Text) { door = it }
                }
            }
            FieldLabel("联系人")
            AddrInput(contact, "联系人姓名", KeyboardType.Text) { contact = it; err = "" }
            FieldLabel("联系电话")
            AddrInput(phone, "留空则使用账户手机号", KeyboardType.Phone) { phone = it; err = "" }

            if (err.isNotBlank()) Text(err, fontSize = 12.sp, color = Palette.err,
                modifier = Modifier.padding(top = 4.dp))

            Spacer(Modifier.height(12.dp))
            SubmitButton(
                text = if (initial == null) "保存地址" else "保存修改",
                enabled = community.isNotBlank() && contact.isNotBlank() && phoneOk(phone),
            ) {
                val payload = JSONObject()
                    .put("community", community.trim())
                    .put("building", building.trim())
                    .put("door", door.trim())
                    .put("contact", contact.trim())
                    .put("phone", phone.trim())
                onSubmit(payload)
            }
            Spacer(Modifier.height(8.dp))
            TextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) { Text("取消") }
            Spacer(Modifier.height(8.dp))
        }
    }
}

@Composable
private fun SheetHeader(title: String, onClose: () -> Unit) {
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
private fun AddrInput(value: String, placeholder: String, keyboard: KeyboardType, onChange: (String) -> Unit) {
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
private fun SubmitButton(text: String, enabled: Boolean, onClick: () -> Unit) {
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

private fun phoneOk(phone: String): Boolean {
    if (phone.isBlank()) return true // 后端会用账户手机兜底
    return Regex("^1\\d{10}$").matches(phone.trim())
}
