package com.ymm.boss.user.page

import android.content.Context
import com.ymm.boss.user.api.LocationProvider
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

// doLocate 取位协程;LocateAction 卡片见 LocateAction.kt(上一会话拆出)。

/** 取当前位置:成功把坐标串预填门牌号参考;Failure 三态给可读文案,不静默清空。 */
internal fun doLocate(
    context: Context,
    scope: CoroutineScope,
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
            // 区分 Failure 三态：拒绝 / 不可用 / 超时（真机 102 联调发现静默清空像按钮坏了）。
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
