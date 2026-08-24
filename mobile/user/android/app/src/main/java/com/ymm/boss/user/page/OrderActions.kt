package com.ymm.boss.user.page

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import kotlinx.coroutines.launch
import org.json.JSONObject

// 订单详情底部固定操作栏(按状态分派动作) + 取消确认弹窗 + 催单/拨号副作用。

@Composable
internal fun BoxScope.FloatingActionBar(nav: Nav, no: String, order: JSONObject?, status: String) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    Surface(
        tonalElevation = 2.dp,
        color = Palette.panel,
        modifier = Modifier.fillMaxWidth().align(Alignment.BottomCenter),
    ) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            when (status) {
                "PENDING", "RESERVED", "INSTALLING" -> {
                    Button(
                        onClick = { scope.launch { urge(ctx, no) } },
                        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                        modifier = Modifier.weight(1f).height(44.dp),
                    ) { Text("催单", fontSize = 15.sp, fontWeight = FontWeight.W600, color = Color.White) }
                    OutlinedButton(
                        onClick = { scope.launch { dialTechnician(ctx, no) } },
                        modifier = Modifier.weight(1f).height(44.dp),
                    ) { Text("联系师傅", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
                    OutlinedButton(
                        onClick = { nav.push(Route.OrderChangeAddress(no)) },
                        modifier = Modifier.weight(1f).height(44.dp),
                    ) { Text("变更地址", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
                }
                "DONE" -> {
                    if (order?.optBoolean("canRate") == true) {
                        Button(
                            onClick = { nav.push(Route.Rate(no)) },
                            colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                            modifier = Modifier.weight(1f).height(44.dp),
                        ) { Text("去评价", fontSize = 15.sp, fontWeight = FontWeight.W600, color = Color.White) }
                    } else {
                        Box(
                            Modifier.weight(1f).background(Palette.success.copy(alpha = 0.08f), RoundedCornerShape(8.dp)),
                            contentAlignment = Alignment.Center,
                        ) { Text("订单已完成,感谢您的选择!", fontSize = 13.sp, color = Palette.success) }
                    }
                    OutlinedButton(
                        onClick = { scope.launch { dialTechnician(ctx, no) } },
                        modifier = Modifier.weight(1f).height(44.dp),
                    ) { Text("联系师傅", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
                }
                "CANCELLED" -> {
                    Box(
                        Modifier.weight(1f).height(44.dp)
                            .background(Palette.err.copy(alpha = 0.08f), RoundedCornerShape(8.dp)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                Icons.Filled.Cancel,
                                contentDescription = null,
                                tint = Palette.err,
                                modifier = Modifier.size(16.dp),
                            )
                            Spacer(Modifier.width(6.dp))
                            Text(
                                "订单已取消",
                                fontSize = 13.sp,
                                fontWeight = FontWeight.W500,
                                color = Palette.err,
                            )
                        }
                    }
                    OutlinedButton(
                        onClick = { nav.push(Route.Complaint) },
                        modifier = Modifier.weight(1f).height(44.dp),
                    ) { Text("联系客服", fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.primary) }
                }
            }
        }
    }
}

private suspend fun urge(ctx: Context, no: String) {
    try {
        OrderApi.urge(no)
        Toast.makeText(ctx, "已通知师傅加紧处理", Toast.LENGTH_SHORT).show()
    } catch (e: Exception) {
        Toast.makeText(ctx, "催单失败,请稍后重试", Toast.LENGTH_SHORT).show()
    }
}

// 装维师傅明文电话走 GET orders/{orderNo}/technician-contact,取到后拉起拨号盘。
internal suspend fun dialTechnician(ctx: Context, no: String) {
    try {
        val phone = OrderApi.technicianContact(no).optString("phone")
        if (phone.isBlank()) throw Api.HttpError(404, "empty phone")
        ctx.startActivity(Intent(Intent.ACTION_DIAL, Uri.parse("tel:$phone")))
    } catch (e: Exception) {
        Toast.makeText(ctx, "暂无法获取师傅电话,请稍后重试", Toast.LENGTH_SHORT).show()
    }
}

@Composable
internal fun CancelDialog(nav: Nav, no: String, onDismiss: () -> Unit) {
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("确认取消该订单?") },
        text = { Text("取消后将释放预占端口并关闭订单。") },
        confirmButton = {
            TextButton(onClick = {
                scope.launch {
                    try {
                        OrderApi.cancel(no)
                        Toast.makeText(ctx, "订单已取消", Toast.LENGTH_SHORT).show()
                        onDismiss()
                        nav.pop()
                    } catch (e: Exception) {
                        Toast.makeText(ctx, "取消失败,请稍后重试", Toast.LENGTH_SHORT).show()
                        onDismiss()
                    }
                }
            }) { Text("确认取消", color = Palette.err) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("再想想", color = Palette.muted) } },
    )
}
