package com.ymm.boss.worker

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.ui.AppRoot
import com.ymm.boss.worker.ui.theme.WorkerTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Api.init(this)
        setContent {
            WorkerTheme {
                AppRoot(loggedIn = Api.token().isNotEmpty())
            }
        }
    }
}
