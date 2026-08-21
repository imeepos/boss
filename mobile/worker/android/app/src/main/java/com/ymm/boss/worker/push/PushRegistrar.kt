package com.ymm.boss.worker.push

import android.content.Context
import cn.jpush.android.api.JPushInterface
import com.ymm.boss.worker.api.Api
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.json.JSONObject

// 推送注册:RegistrationID 就绪后上报 BOSS(POST /push/device,契约 worker/misc.yaml)。
// RegistrationID 在 JPush init 后异步到达,轮询重试;同 token+rid 只报一次(prefs 记忆)。
object PushRegistrar {
    private const val PREFS = "boss_worker_push"
    private const val KEY_UPLOADED = "uploaded_rid_token"
    private const val POLL_TIMES = 10
    private const val POLL_INTERVAL_MS = 3_000L
    private val scope = CoroutineScope(Dispatchers.IO)

    /** ensureRegisteredOnLogin 登录成功时调用(doLogin 无 context,经 Api 取全局)。 */
    fun ensureRegisteredOnLogin() = ensureRegistered(Api.context())

    /** ensureRegistered 登录成功/启动且已登录时调用;未登录不上报(服务端按主体绑定)。 */
    fun ensureRegistered(appContext: Context) {
        if (Api.token().isEmpty()) return
        scope.launch {
            val rid = awaitRegistrationID(appContext)
            if (rid.isEmpty()) return@launch
            val state = "$rid|${Api.token()}"
            val prefs = appContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            if (prefs.getString(KEY_UPLOADED, "") == state) return@launch
            try {
                Api.post("/push/device", JSONObject().put("registrationId", rid).put("vendor", "jpush"))
                prefs.edit().putString(KEY_UPLOADED, state).apply()
            } catch (_: Exception) {
                // 尽力而为:下次登录/启动重试
            }
        }
    }

    /** awaitRegistrationID 轮询取 RegistrationID;超时返回空串。 */
    private suspend fun awaitRegistrationID(ctx: Context): String {
        repeat(POLL_TIMES) {
            val rid = JPushInterface.getRegistrationID(ctx) ?: ""
            if (rid.isNotEmpty()) return rid
            delay(POLL_INTERVAL_MS)
        }
        return JPushInterface.getRegistrationID(ctx) ?: ""
    }
}
