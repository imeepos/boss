package com.ymm.boss.user.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember

/**
 * 表单防重复提交守卫(弱网最小保障版三横切点之一,2026-08-27 上线计划 D-3)。
 * 弱网下用户反复点提交会造成重复下单/重复充值,本守卫在异步提交完成前拦截再次触发。
 *
 * 用法:
 *   val guard = rememberSubmitGuard()
 *   Button(enabled = !guard.active, onClick = {
 *       if (!guard.acquire()) return@Button            // 提交中,拦截
 *       scope.launch { try { submit() } finally { guard.release() } }  // 完成后再复位
 *   }) { Text(if (guard.active) "提交中…" else "确认提交") }
 */
class SubmitGuard internal constructor(private val state: MutableState<Boolean>) {
    val active: Boolean get() = state.value

    /** 抢占提交权;已提交中返回 false。 */
    fun acquire(): Boolean {
        if (state.value) return false
        state.value = true
        return true
    }

    /** 提交结束(成功/失败均需调用)后复位。 */
    fun release() {
        state.value = false
    }
}

@Composable
fun rememberSubmitGuard(): SubmitGuard = remember { SubmitGuard(mutableStateOf(false)) }