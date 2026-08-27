package com.ymm.boss.user.ui

import androidx.compose.runtime.mutableStateOf
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SubmitGuardTest {

    @Test
    fun `acquire once then blocks until release`() {
        val guard = SubmitGuard(mutableStateOf(false))
        assertFalse(guard.active)
        assertTrue(guard.acquire())
        assertTrue(guard.active)
        assertFalse("提交中不允许二次触发", guard.acquire())
        guard.release()
        assertFalse(guard.active)
        assertTrue("复位后可再次提交", guard.acquire())
    }

    @Test
    fun `release is idempotent`() {
        val guard = SubmitGuard(mutableStateOf(false))
        guard.release()
        guard.release()
        assertFalse(guard.active)
        assertTrue(guard.acquire())
    }
}