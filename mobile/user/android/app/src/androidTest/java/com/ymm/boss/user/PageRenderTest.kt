package com.ymm.boss.user

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.DeviceConfigurationOverride
import androidx.compose.ui.test.ForcedSize
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.user.page.FaultDetailScreen
import com.ymm.boss.user.page.InvoiceScreen
import com.ymm.boss.user.page.MessagesScreen
import com.ymm.boss.user.page.OrderConfirmScreen
import com.ymm.boss.user.page.ProductScreen
import com.ymm.boss.user.page.ProductsScreen
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

    @Test
    fun orderConfirmShowsCardsAndPickTitle() {
        compose.setContent { OrderConfirmScreen(Nav(Route.OrderConfirm("101")), "101") }
        // 套餐信息 + 安装地址选择两张卡标题必现;无网络数据时地址列表为空提示
        compose.onNodeWithText("套餐信息").assertIsDisplayed()
        compose.onNodeWithText("选择安装地址").assertIsDisplayed()
    }

    @Test
    fun productsShowsCategoryCaps() {
        // 服务 tab 渲染冒烟:分类胶囊与搜索框与网络无关恒渲染(数据态为骨架屏/列表/空态)
        compose.setContent { ProductsScreen(Nav(Route.Products)) }
        compose.onNodeWithText("宽带").assertIsDisplayed()
        compose.onNodeWithText("5G").assertIsDisplayed()
        compose.onNodeWithText("增值服务").assertIsDisplayed()
    }

    @Test
    fun invoiceShowsCardsAndTerminals() {
        // 发票页(佳宁冒烟项)骨架:标题与卡片标题与网络无关恒渲染;空数据态有明示空态/占位
        compose.setContent { InvoiceScreen(Nav(Route.Invoice)) }
        compose.onNodeWithText("电子发票").assertIsDisplayed()
        compose.onNodeWithText("开票信息").assertIsDisplayed()
        compose.onNodeWithText("可开票账期").assertIsDisplayed()
    }

    /**
     * 360dp 窄屏基线:防 PillTab 类水平溢出回归(2026-08-21 曾 5 胶囊在 360dp 溢出)。
     * 用 DeviceConfigurationOverride.ForcedSize 强制 360x800 视口,断言骨架关键节点
     * 仍可显示;不依赖网络数据。ForcedSize 兼容 ui-test 1.7~1.11(Width 是 1.9+ 新 API)。
     */
    @Test
    fun messagesSkeletonFitsNarrow360dpViewport() {
        compose.setContent {
            DeviceConfigurationOverride(
                DeviceConfigurationOverride.ForcedSize(DpSize(360.dp, 800.dp)),
            ) {
                MessagesScreen(Nav(Route.Messages))
            }
        }
        compose.onNodeWithText("消息中心").assertIsDisplayed()
        compose.onNodeWithText("全部消息").assertIsDisplayed()
        compose.onNodeWithText("未读消息").assertIsDisplayed()
    }
}
