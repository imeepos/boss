package com.ymm.boss.worker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.ui.AppRoot
import com.ymm.boss.worker.ui.theme.BgArgb
import com.ymm.boss.worker.ui.theme.WorkerTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        // 状态栏与主题同色(AppRoot 全屏铺 Bg,Bar 沉浸其下);App 仅浅色主题,强制深色图标
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.light(BgArgb, BgArgb),
        )
        setContent {
            WorkerTheme {
                AppRoot(loggedIn = Api.token().isNotEmpty())
            }
        }
    }
}
