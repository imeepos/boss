package com.ymm.boss.user.page

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.ymm.boss.user.api.UpdateApi

/**
 * 在线更新弹框(需求:新版本弹框提醒、可忽略不强制;force 仅来自服务端强升线)。
 * UpdateGate 挂 AppRoot 做启动静默检查,每次进程只查一次。
 */
@Composable
fun UpdateGate() {
    var info by remember { mutableStateOf<UpdateApi.UpdateInfo?>(null) }
    var checked by remember { mutableStateOf(false) }
    val context = LocalContext.current
    LaunchedEffect(Unit) {
        if (!checked) {
            checked = true
            info = UpdateApi.check(context)?.takeIf { it.updateAvailable }
        }
    }
    UpdateDialog(info = info, onDismiss = { info = null })
}

/** 更新弹框;manual=true 为设置页手动检查(可能提示"已是最新")。 */
@Composable
fun UpdateDialog(info: UpdateApi.UpdateInfo?, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val cur = info ?: return
    AlertDialog(
        onDismissRequest = { if (!cur.force) onDismiss() },
        title = { Text("发现新版本 v${cur.version}") },
        text = {
            Text(
                buildString {
                    append(cur.notes.ifBlank { "优化体验,修复已知问题。" })
                    if (cur.sizeMb.isNotEmpty()) append("\n\n安装包大小:${cur.sizeMb}")
                    if (cur.force) append("\n\n当前版本过低,需更新后继续使用。")
                },
            )
        },
        confirmButton = {
            TextButton(onClick = {
                UpdateApi.openDownload(context, cur)
                if (!cur.force) onDismiss()
            }) { Text("立即更新") }
        },
        dismissButton = {
            if (!cur.force) TextButton(onClick = onDismiss) { Text("忽略") }
        },
    )
}
