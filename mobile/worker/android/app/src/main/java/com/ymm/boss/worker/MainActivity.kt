package com.ymm.boss.worker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowInsetsControllerCompat
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.ui.AppRoot
import com.ymm.boss.worker.ui.theme.StatusBarSolidArgb
import com.ymm.boss.worker.ui.theme.WorkerTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        // 状态栏区域由 AppRoot 统一画固定纯色带(与头部渐变起点同源,对齐 user 端方案),图标白色。
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(StatusBarSolidArgb),
        )
        WindowInsetsControllerCompat(window, window.decorView).isAppearanceLightStatusBars = false
        setContent {
            WorkerTheme {
                AppRoot(loggedIn = Api.token().isNotEmpty())
            }
        }
    }
}
