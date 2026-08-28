package com.ymm.boss.user

import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.user.api.Api
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith

/**
 * BUG-02 会话失效策略测试(纯逻辑、无网络):
 * 401(HTTP 层,凭证真失效)清除会话;40100(信封业务校验失败,如验证码错/旧口令错)不得清除;
 * 短窗口冷却下 onUnauthorized 只播一次。
 */
@RunWith(AndroidJUnit4::class)
class ApiAuthPolicyTest {

    @Test
    fun http401ClearsTokenAndNotifies() {
        Api.init(ApplicationProvider.getApplicationContext())
        var notified = 0
        Api.onUnauthorized = { notified++ }
        Api.setToken("tk-valid")

        Api.handleUnauthorized(401, "/profile/security/password")

        assertEquals("HTTP 401 应清除 token", "", Api.token())
        assertEquals("HTTP 401 应播报一次", 1, notified)
        Api.onUnauthorized = null
    }

    @Test
    fun envelope40100DoesNotClearToken() {
        Api.init(ApplicationProvider.getApplicationContext())
        var notified = 0
        Api.onUnauthorized = { notified++ }
        Api.setToken("tk-valid")

        Api.handleUnauthorized(40100, "/auth/login")

        assertEquals("信封 40100 不得清除 token(业务校验失败非凭证失效)", "tk-valid", Api.token())
        assertEquals("信封 40100 不得播报登录失效", 0, notified)
        Api.onUnauthorized = null
    }

    @Test
    fun cooldownDedupsNotificationInWindow() {
        Api.init(ApplicationProvider.getApplicationContext())
        var notified = 0
        Api.onUnauthorized = { notified++ }
        Api.setToken("tk-valid")

        Api.handleUnauthorized(401, "/a")
        Api.handleUnauthorized(401, "/b") // 3s 冷却窗口内并入,不重复播报

        assertEquals("冷却窗口内只播报一次", 1, notified)
        assertEquals("重复 401 幂等清除", "", Api.token())
        Api.onUnauthorized = null
    }
}