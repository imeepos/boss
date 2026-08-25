package com.ymm.boss.worker.ui

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
import androidx.compose.ui.res.stringResource
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.UpdateApi

/**
 * 在线更新弹框(需求:新版本弹框提醒、可忽略不强制;force 仅来自服务端强升线)。
 * UpdateGate 挂 AppRoot 做启动静默检查,每进程一次;与用户端 UpdateDialog 同构。
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

@Composable
fun UpdateDialog(info: UpdateApi.UpdateInfo?, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val ctx = context
    val cur = info ?: return
    AlertDialog(
        onDismissRequest = { if (!cur.force) onDismiss() },
        title = { Text(stringResource(R.string.upd_available, cur.version)) },
        text = {
            Text(
                buildString {
                    append(cur.notes.ifBlank { stringResource(R.string.upd_optimize) })
                    if (cur.sizeMb.isNotEmpty()) append(ctx.getString(R.string.upd_size_fmt, cur.sizeMb))
                    if (cur.force) append(ctx.getString(R.string.upd_force_fmt, ""))
                },
            )
        },
        confirmButton = {
            TextButton(onClick = {
                UpdateApi.openDownload(context, cur)
                if (!cur.force) onDismiss()
            }) { Text(stringResource(R.string.upd_now)) }
        },
        dismissButton = {
            if (!cur.force) TextButton(onClick = onDismiss) { Text(stringResource(R.string.upd_ignore)) }
        },
    )
}
