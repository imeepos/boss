package com.ymm.boss.user.page

import com.ymm.boss.user.api.Api
import org.junit.Assert.assertEquals
import org.junit.Test
import java.io.IOException

class LoginErrorMessageTest {

    @Test
    fun `40100 differs by login mode`() {
        assertEquals(
            "验证码错误或已过期，请重新获取",
            loginErrorMessage(Api.HttpError(40100, "未认证或凭证无效"), "sms"),
        )
        assertEquals(
            "手机号或密码不正确",
            loginErrorMessage(Api.HttpError(40100, "未认证或凭证无效"), "password"),
        )
    }

    @Test
    fun `envelope codes map to readable text`() {
        assertEquals("账号不存在，请先注册", Api.friendlyMessage(Api.HttpError(40400, "not found")))
        assertEquals("输入格式不正确，请检查后重填", Api.friendlyMessage(Api.HttpError(42200, "invalid")))
        assertEquals("服务开小差了，请稍后重试", Api.friendlyMessage(Api.HttpError(50000, "internal")))
        assertEquals("服务暂不可用，请稍后重试", Api.friendlyMessage(Api.HttpError(50200, "downstream")))
    }

    @Test
    fun `transport errors map to network text`() {
        assertEquals("网络连接失败，请检查网络", Api.friendlyMessage(IOException("timeout")))
        assertEquals("服务器异常，请稍后重试", Api.friendlyMessage(Api.HttpError(503, "HTTP 503")))
        assertEquals("登录状态已失效，请重新登录", Api.friendlyMessage(Api.HttpError(401, "HTTP 401")))
    }

    @Test
    fun `unknown envelope code falls back to server message`() {
        assertEquals("自定义业务错误", Api.friendlyMessage(Api.HttpError(41800, "自定义业务错误")))
    }
}
