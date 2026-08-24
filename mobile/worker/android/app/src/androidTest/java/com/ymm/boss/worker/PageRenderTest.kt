package com.ymm.boss.worker

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.worker.ui.NavHost
import com.ymm.boss.worker.ui.NaviScreen
import com.ymm.boss.worker.ui.Screen
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * worker 端纯渲染冒烟(首个 androidTest):
 * 至少一个用例落在设备/模拟器上,配合 after-connected-test.sh 的 tests>0 断言,
 * 防"有 runner 没用例"的空跑回归(user 端 2026-08-24 实踩)。
 */
@RunWith(AndroidJUnit4::class)
class PageRenderTest {

    @get:Rule
    val compose = createComposeRule()

    @Test
    fun naviShowsKeysAndNoCrash() {
        compose.setContent { NaviScreen(NavHost(Screen.Navi("TKT-001")), "TKT-001") }
        // 一键导航页:静态字段应渲染,页面不应 crash(工单号 404 时也有空态兜底)
        compose.onNodeWithText("一键导航").assertIsDisplayed()
    }
}