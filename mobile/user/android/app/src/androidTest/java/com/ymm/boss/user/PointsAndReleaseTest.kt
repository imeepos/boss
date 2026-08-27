package com.ymm.boss.user

import android.content.Context
import android.content.pm.PackageManager
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.user.page.PointsScreen
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Route
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * 积分页与发布基建(2026-08-27 D0/D1-D4)冒烟:
 * 1) PointsScreen 骨架与各卡标题必现(与网络无关,离线/在线均渲染);
 * 2) Manifest 已声明 POST_NOTIFICATIONS(Android 13+ 运行时权限立项)。
 * 仅能 connectedDebugAndroidTest 执行,本地 JVM 无法运行。
 */
@RunWith(AndroidJUnit4::class)
class PointsAndReleaseTest {

    @get:Rule
    val compose = createComposeRule()

    @Test
    fun pointsPageShowsOverviewSections() {
        compose.setContent { PointsScreen(Nav(Route.Points)) }
        compose.onNodeWithText("我的积分").assertIsDisplayed()
        compose.onNodeWithText("积分余额").assertIsDisplayed()
        compose.onNodeWithText("积分兑换").assertIsDisplayed()
        compose.onNodeWithText("赚积分").assertIsDisplayed()
        compose.onNodeWithText("积分明细").assertIsDisplayed()
    }

    @Test
    fun manifestDeclaresPostNotifications() {
        val pm = ApplicationProvider.getApplicationContext<Context>().packageManager
        val info = pm.getPackageInfo("com.ymm.boss.user", PackageManager.GET_PERMISSIONS)
        assertTrue(info.requestedPermissions.contains("android.permission.POST_NOTIFICATIONS"))
    }
}