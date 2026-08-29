package com.ymm.boss.worker.ui

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
 * worker 端 TopBar 契约 instrumented 测试。
 * 契约出处: docs/design/topbar-contract.md —— 两端共享签名
 * TopBar(title, onBack?, action?, onAction?) 与行为语义;
 * 本测试锁定签名完整性与交互语义(返回/动作回调、可选参数省略),
 * 视觉参数(渐变背景/字号/间距)由契约文档人工锁定,不在断言范围。
 */
@RunWith(AndroidJUnit4::class)
class TopBarContractTest {

    @get:Rule
    val compose = createComposeRule()

    // 契约 §1:四参数签名下,标题/返回键/右动作三区齐渲染
    @Test
    fun fullSignatureRendersAllParts() {
        compose.setContent {
            TopBar("契约标题", onBack = {}, action = "动作", onAction = {})
        }
        compose.onNodeWithText("契约标题").assertIsDisplayed()
        compose.onNodeWithText("<").assertIsDisplayed()
        compose.onNodeWithText("动作").assertIsDisplayed()
    }

    // 契约 §1:返回键点击触发 onBack
    @Test
    fun backKeyClickInvokesOnBack() {
        var backCalled = false
        compose.setContent {
            TopBar("契约标题", onBack = { backCalled = true })
        }
        compose.onNodeWithText("<").performClick()
        assertTrue(backCalled)
    }

    // 契约 §1:右动作点击触发 onAction(onAction 为 null 时是空操作,不属本断言)
    @Test
    fun actionClickInvokesOnAction() {
        var actionCalled = false
        compose.setContent {
            TopBar("契约标题", onBack = {}, action = "动作", onAction = { actionCalled = true })
        }
        compose.onNodeWithText("动作").performClick()
        assertTrue(actionCalled)
    }

    // 契约 §1 行为语义:onBack == null 不渲染返回键
    @Test
    fun nullOnBackOmitsBackKey() {
        compose.setContent {
            TopBar("契约标题", onBack = null)
        }
        compose.onNodeWithText("<").assertDoesNotExist()
    }
}
