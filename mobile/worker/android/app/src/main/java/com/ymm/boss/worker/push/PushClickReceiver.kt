package com.ymm.boss.worker.push

import android.content.Context
import android.content.Intent
import cn.jpush.android.api.NotificationMessage
import cn.jpush.android.service.JPushMessageReceiver
import com.ymm.boss.worker.MainActivity
import org.json.JSONObject

// 通知点击:解析通知 extras 里的工单号,拉起 MainActivity(singleTop)并携带深链参数。
// 官方要求 JPushMessageReceiver 子类在 manifest 声明(无需 intent-filter),SDK 内部派发。
class PushClickReceiver : JPushMessageReceiver() {
    override fun onNotifyMessageOpened(context: Context, message: NotificationMessage) {
        val no = ticketNoFrom(message.notificationExtras)
        // 冷启动场景 AppRoot 尚未组合,先记 pending;热启动走 intent extra 双保险。
        DeepLink.request(no)
        val intent = Intent(context, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        if (no != null) intent.putExtra(DeepLink.EXTRA_TICKET_NO, no)
        context.startActivity(intent)
    }
}

// extras 为 JSON 字符串,工单号键名按服务端推送契约优先 ticket_no,兼容历史别名
private fun ticketNoFrom(extras: String?): String? {
    if (extras.isNullOrBlank()) return null
    val obj = runCatching { JSONObject(extras) }.getOrNull() ?: return null
    return listOf("ticket_no", "ticketNo", "orderNo", "no")
        .firstNotNullOfOrNull { obj.optString(it).takeIf(String::isNotBlank) }
}
