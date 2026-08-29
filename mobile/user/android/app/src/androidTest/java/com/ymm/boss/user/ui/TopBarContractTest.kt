package com.ymm.boss.user.ui

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * TopBar 契约 instrumented 测试,契约出处 docs/design/topbar-contract.md。
 * 锁定签名与交互语义:三要素(标题/返回键/动作)渲染、onBack/onAction 回调、可选参数省略语义。
 * 视觉参数(48dp 高度/statusBarSolid 纯色/字号)由契约文档人工锁定,不在本测试断言范围。
 * 仅能在设备/CI 模拟器上执行(connectedDebugAndroidTest),本地 JVM 无法运行。
 */
@RunWith(AndroidJUnit4::class)
class TopBarContractTest {

    @get:Rule
    val compose = createComposeRule()

    @Test
    fun signatureRendersTitleBackAndAction() {
        compose.setContent { TopBar("契约标题", onBack = {}, action = "动作", onAction = {}) }
        compose.onNodeWithText("契约标题").assertIsDisplayed()
        compose.onNodeWithText("‹").assertIsDisplayed()
        compose.onNodeWithText("动作").assertIsDisplayed()
    }

    @Test
    fun backClickInvokesOnBack() {
        var backCalled = false
        compose.setContent { TopBar("契约标题", onBack = { backCalled = true }) }
        compose.onNodeWithText("‹").performClick()
        assertTrue(backCalled)
    }

    @Test
    fun actionClickInvokesOnAction() {
        var actionCalled = false
        compose.setContent { TopBar("契约标题", action = "动作", onAction = { actionCalled = true }) }
        compose.onNodeWithText("动作").performClick()
        assertTrue(actionCalled)
    }

    @Test
    fun optionalParamsOmittedRenderNothing() {
        // 契约 §1 行为语义:onBack==null 不渲染返回键,action==null 不渲染右侧动作。
        compose.setContent { TopBar("契约标题") }
        compose.onNodeWithText("契约标题").assertIsDisplayed()
        compose.onNodeWithText("‹").assertDoesNotExist()
        compose.onNodeWithText("动作").assertDoesNotExist()
    }
}
