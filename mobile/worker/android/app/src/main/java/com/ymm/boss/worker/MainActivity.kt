package com.ymm.boss.worker

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowInsetsControllerCompat
import androidx.activity.result.contract.ActivityResultContracts
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.push.DeepLink
import com.ymm.boss.worker.push.PushRegistrar
import com.ymm.boss.worker.ui.AppRoot
import com.ymm.boss.worker.ui.theme.StatusBarSolidArgb
import com.ymm.boss.worker.ui.theme.WorkerTheme

class MainActivity : ComponentActivity() {
    private val locationPermission = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        requestLocationPermission()
        takeDeepLink(intent)
        // 已登录则补报 RegistrationID(登录成功那次未上报成功/换设备场景)。
        PushRegistrar.ensureRegistered(applicationContext)
        // 崩溃留痕启动补传(未登录静默跳过,内部 IO 协程)。
        com.ymm.boss.worker.util.CrashLog.uploadPending(applicationContext)
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

    private fun requestLocationPermission() {
        val permissions = mutableListOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) permissions += Manifest.permission.POST_NOTIFICATIONS
        if (permissions.any { checkSelfPermission(it) != PackageManager.PERMISSION_GRANTED }) locationPermission.launch(permissions.toTypedArray())
    }

    private fun takeDeepLink(intent: Intent?) {
        DeepLink.request(intent?.getStringExtra(DeepLink.EXTRA_TICKET_NO))
    }
}
