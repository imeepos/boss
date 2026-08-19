package com.ymm.boss.user

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
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
        setContent { BossTheme { AppRoot() } }
    }
}

@Composable
fun AppRoot() {
    val nav = remember { Nav(if (Api.token().isNotEmpty()) Route.Home else Route.Login) }
    Surface(Modifier.fillMaxSize()) {
        PageScaffold(nav, tabKeyOf(nav.current), nav.current !is Route.Login) {
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
