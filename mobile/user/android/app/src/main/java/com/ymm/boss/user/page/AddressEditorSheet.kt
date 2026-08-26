package com.ymm.boss.user.page

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.LocationProvider
import com.ymm.boss.user.ui.Palette
import kotlinx.coroutines.launch
import org.json.JSONObject

/**
 * 新增/编辑地址底部弹窗。
 * 顶部"使用当前位置"按钮：拿 GPS → 把"GPS: N, E"预填到门牌号（可改）。
 * 小区输入带历史下拉（recentCommunities 调用方注入），减少重复输入。
 * 真机拒绝过一次定位权限后，再次点按钮会先弹 rationale 对话框解释用途，
 * 用户同意后再走 permissionLauncher（否则系统不再弹权限框）。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddressEditorSheet(
    initial: JSONObject?,
    recentCommunities: List<String>,
    onDismiss: () -> Unit,
    onSubmit: (JSONObject) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val activity = context as? android.app.Activity

    var community by remember { mutableStateOf(initial?.optString("community").orEmpty()) }
    var building by remember { mutableStateOf(initial?.optString("building").orEmpty()) }
    var door by remember { mutableStateOf(initial?.optString("door").orEmpty()) }
    var contact by remember { mutableStateOf(initial?.optString("contact").orEmpty()) }
    var phone by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    var locationHint by remember { mutableStateOf("") }
    var locating by remember { mutableStateOf(false) }
    var rationaleVisible by remember { mutableStateOf(false) }
    val communityOptions = remember(recentCommunities, community) {
        recentCommunities.filter { it.isNotBlank() && it != community }.distinct()
    }

    fun launchLocation() = doLocate(
        context, scope,
        setLocating = { locating = it },
        setHint = { locationHint = it },
        setDoor = { door = it },
        setErr = { err = it },
    )

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { granted ->
        val ok = granted[Manifest.permission.ACCESS_FINE_LOCATION] == true ||
                granted[Manifest.permission.ACCESS_COARSE_LOCATION] == true
        if (ok) launchLocation() else { err = "未授予定位权限"; locationHint = "" }
    }

    fun startPermissionFlow() {
        // Activity 上报：true 表示用户拒绝过且未勾"不再询问"；此时直接再 launch
        // 系统不会再弹窗，必须先 rationale。
        if (activity != null && activity.shouldShowRequestPermissionRationale(
                Manifest.permission.ACCESS_FINE_LOCATION)) {
            rationaleVisible = true
        } else {
            permissionLauncher.launch(arrayOf(
                Manifest.permission.ACCESS_FINE_LOCATION,
                Manifest.permission.ACCESS_COARSE_LOCATION,
            ))
        }
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Palette.panel,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
            SheetHeader(title = if (initial == null) "新增家庭地址" else "编辑家庭地址", onClose = onDismiss)
            Spacer(Modifier.height(4.dp))
            LocateAction(
                locating = locating,
                hint = locationHint,
                onClick = {
                    err = ""
                    if (LocationProvider.hasPermission(context)) launchLocation()
                    else startPermissionFlow()
                },
            )
            Spacer(Modifier.height(8.dp))
            FieldLabel("小区 / 楼盘")
            CommunityField(community, communityOptions) { community = it; err = "" }
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

    if (rationaleVisible) {
        AlertDialog(
            onDismissRequest = { rationaleVisible = false },
            title = { Text("需要定位权限") },
            text = { Text("获取当前位置用于把经纬度填入门牌号参考，地址仍可手动修改。") },
            confirmButton = {
                TextButton(onClick = {
                    rationaleVisible = false
                    permissionLauncher.launch(arrayOf(
                        Manifest.permission.ACCESS_FINE_LOCATION,
                        Manifest.permission.ACCESS_COARSE_LOCATION,
                    ))
                }) { Text("继续") }
            },
            dismissButton = {
                TextButton(onClick = { rationaleVisible = false }) { Text("暂不开启") }
            },
        )
    }
}

private fun doLocate(
    context: android.content.Context,
    scope: kotlinx.coroutines.CoroutineScope,
    setLocating: (Boolean) -> Unit,
    setHint: (String) -> Unit,
    setDoor: (String) -> Unit,
    setErr: (String) -> Unit,
) {
    setLocating(true)
    scope.launch {
        try {
            val p = LocationProvider.current(context)
            val s = LocationProvider.format(p)
            setHint("当前位置：$s（已填入门牌号，可修改）")
            setDoor("GPS: $s")
            setErr("")
        } catch (e: LocationProvider.Failure) {
            // 区分 Failure 三态：拒绝 / 不可用 / 超时，给用户可读反馈而不是静默清空。
            // 真机 102 联调发现无定位设备静默清空 hint+door 用户没反馈，会怀疑按钮坏了。
            setHint("")
            setDoor("")
            setErr(e.message ?: "定位失败，请重试")
        } catch (_: Exception) {
            setHint("")
            setDoor("")
            setErr("定位失败，请重试")
        } finally {
            setLocating(false)
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
private fun CommunityField(value: String, options: List<String>, onChange: (String) -> Unit) {
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
    if (phone.isBlank()) return true
    return Regex("^1\\d{10}$").matches(phone.trim())
}

@Composable
private fun FieldLabel(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.muted, modifier = Modifier.padding(bottom = 4.dp))
}