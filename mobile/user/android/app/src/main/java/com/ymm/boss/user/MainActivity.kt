package com.ymm.boss.user

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowInsetsControllerCompat
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.ui.BossTheme
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.PageScaffold
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.RouteScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        enableEdgeToEdge()
        // 系统栏保持透明 edge-to-edge:首页渐变延伸到状态栏后方,其余页顶部为蓝色 TopBar
        WindowInsetsControllerCompat(window, window.decorView).isAppearanceLightStatusBars = false
        setContent { BossTheme { AppRoot() } }
    }
}

@Composable
fun AppRoot() {
    val nav = remember { Nav(if (Api.token().isNotEmpty()) Route.Home else Route.Login) }
    Surface(Modifier.fillMaxSize()) {
        val key = tabKeyOf(nav.current)
        // 首页自带 Scaffold+NavigationBar 底栏,PageScaffold 不再重复渲染;
        // 首页顶部渐变需延伸到状态栏后方,顶层不再吃掉顶部 insets
        val isHome = nav.current == Route.Home
        val showTabs = key.isNotEmpty() && !isHome
        PageScaffold(nav, key, showTabs, extendIntoStatusBar = isHome) {
            RouteScreen(nav.current, nav)
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
