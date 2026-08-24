package com.ymm.boss.worker

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowInsetsControllerCompat
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.push.DeepLink
import com.ymm.boss.worker.push.PushRegistrar
import com.ymm.boss.worker.ui.AppRoot
import com.ymm.boss.worker.ui.theme.StatusBarSolidArgb
import com.ymm.boss.worker.ui.theme.WorkerTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        takeDeepLink(intent)
        // 已登录则补报 RegistrationID(登录成功那次未上报成功/换设备场景)。
        PushRegistrar.ensureRegistered(applicationContext)
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

    // singleTop:通知点击拉起已存活 Activity 走 onNewIntent 而非重建
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        takeDeepLink(intent)
    }

    private fun takeDeepLink(intent: Intent?) {
        DeepLink.request(intent?.getStringExtra(DeepLink.EXTRA_TICKET_NO))
    }
}
