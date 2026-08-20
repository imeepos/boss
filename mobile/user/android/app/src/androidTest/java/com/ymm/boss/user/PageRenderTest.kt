package com.ymm.boss.user

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.user.page.FaultDetailScreen
import com.ymm.boss.user.page.ProductScreen
import com.ymm.boss.user.page.ReceiptScreen
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Route
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * 纯渲染冒烟:断言页面骨架与按钮文案。
 * 仅能在设备/CI 模拟器上执行(connectedDebugAndroidTest),本地 JVM 无法运行。
 */
@RunWith(AndroidJUnit4::class)
class PageRenderTest {

    @get:Rule
    val compose = createComposeRule()

    @Test
    fun faultDetailShowsSkeletonAndActions() {
        compose.setContent { FaultDetailScreen(Nav(Route.FaultDetail("TKT-001")), "TKT-001") }
        compose.onNodeWithText("报修详情").assertIsDisplayed()
        compose.onNodeWithText("联系师傅").assertIsDisplayed()
        compose.onNodeWithText("催单").assertIsDisplayed()
    }

    @Test
    fun receiptShowsPdfDownloadButton() {
        compose.setContent { ReceiptScreen(Nav(Route.Receipt("PAY-001")), "PAY-001") }
        compose.onNodeWithText("缴费凭证").assertIsDisplayed()
        compose.onNodeWithText("下载凭证(PDF)").assertIsDisplayed()
    }

    @Test
    fun productDetailStacksCardsVertically() {
        compose.setContent { ProductScreen(Nav(Route.Product("101")), "101") }
        compose.onNodeWithText("套餐详情").assertIsDisplayed()
        // AppCard 为 Column:详情/对比/合约三张卡的标题都应同时可见可点
        compose.onNodeWithText("套餐对比").assertIsDisplayed()
        compose.onNodeWithText("合约与说明").assertIsDisplayed()
    }
}
