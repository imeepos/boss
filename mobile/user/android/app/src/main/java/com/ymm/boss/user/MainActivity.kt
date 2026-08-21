package com.ymm.boss.user

import android.os.Bundle
import android.os.Handler
import android.os.Looper
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowInsetsControllerCompat
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
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
        // 状态栏区域由 PageScaffold 统一画固定纯色带(与首页一致,不透明),此处只保证图标为白色。
        WindowInsetsControllerCompat(window, window.decorView).isAppearanceLightStatusBars = false
        setContent { BossTheme { AppRoot() } }
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
    }
}

fun tabKeyOf(route: Route): String = when (route) {
    Route.Home -> "home"
    Route.Products -> "products"
    Route.Orders -> "orders"
    Route.Profile -> "profile"
    else -> ""
}
