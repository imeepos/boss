package com.ymm.boss.user

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.DevModeStore
import com.ymm.boss.user.ui.BossTheme
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.PageRefresh
import com.ymm.boss.user.ui.PageScaffold
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.RouteScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        DevModeStore.init(this)
        enableEdgeToEdge()
        // 上线计划 D0 立项:Android 13+ 通知运行时权限,启动即弹窗(Manifest 已声明)。
        requestNotificationPermission()
        // 状态栏区域由 PageScaffold 统一画固定纯色带(与首页一致,不透明),此处只保证图标为白色。
        WindowInsetsControllerCompat(window, window.decorView).isAppearanceLightStatusBars = false
        setContent { BossTheme { AppRoot() } }
    }

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT < 33) return
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
            == PackageManager.PERMISSION_GRANTED
        ) return
        ActivityCompat.requestPermissions(
            this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1001,
        )
    }
}

@Composable
fun AppRoot() {
    val nav = remember { Nav(if (Api.token().isNotEmpty()) Route.Home else Route.Login) }
    DisposableEffect(nav) {
        val mainHandler = Handler(Looper.getMainLooper())
        Api.onUnauthorized = { mainHandler.post { nav.resetTo(Route.Login) } }
        onDispose { Api.onUnauthorized = null }
    }
    Surface(Modifier.fillMaxSize()) {
        val key = tabKeyOf(nav.current)
        val isHome = nav.current == Route.Home
        val showTabs = key.isNotEmpty() && !isHome
        PageScaffold(nav, key, showTabs) {
            PageRefresh(nav) { RouteScreen(nav.current, nav) }
        }
        // 启动静默检查更新:有新版弹可忽略提醒(fields.md 8F),每进程一次。
        com.ymm.boss.user.page.UpdateGate()
        // 已登录态启动:上报设备注册(B 轨 /push/device,幂等,失败静默)。
        val context = androidx.compose.ui.platform.LocalContext.current
        LaunchedEffect(Unit) {
            if (Api.token().isNotEmpty()) com.ymm.boss.user.api.PushApi.registerDevice(context)
        }
    }
}

fun tabKeyOf(route: Route): String = when (route) {
    Route.Home -> "home"
    Route.Products -> "products"
    Route.Orders -> "orders"
    Route.Points -> "points"
    Route.Profile -> "profile"
    else -> ""
}
