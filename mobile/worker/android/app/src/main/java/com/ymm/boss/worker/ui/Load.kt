package com.ymm.boss.worker.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.State
import androidx.compose.runtime.produceState
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.Api

// 通用异步加载状态(页面数据一律走此包装,失败不白屏)
sealed interface Load<out T> {
    data object Loading : Load<Nothing>
    data class Ok<T>(val data: T) : Load<T>
    data class Fail(val message: String) : Load<Nothing>
}

@Composable
fun <T> loadOnce(vararg keys: Any?, loader: suspend () -> T): State<Load<T>> =
    produceState<Load<T>>(Load.Loading, keys.toList()) {
        value = try {
            Load.Ok(loader())
        } catch (e: Exception) {
            Load.Fail(e.message ?: Api.context().getString(R.string.load_default_fail))
        }
    }
