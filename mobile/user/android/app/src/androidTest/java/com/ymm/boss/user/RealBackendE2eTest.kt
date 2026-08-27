package com.ymm.boss.user

import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.api.DebugApi
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.UserApi
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.runner.RunWith

/**
 * 真实后端关键路径冒烟(佳宁 6 条关键路径, D-1 门禁): 登录真码 → 产品浏览 → 实名读 → 订单读 → 账单读。
 * 流程: 发码(https 认证流转真码)→ 经 /debug/sms-code 取真码(102 dev 模式)→ 登录拿 token → 各读路径断言。
 * 依赖: emulator 可达 192.168.0.102;102 BOSS_DEBUG_SMS 开启(否则 Assume 跳过,不误报)。
 * 测试号唯一事实源: .agents/skills/bossctl-cli/test-accounts.json customers[0](13900001234, VERIFIED, 有真实订单)。
 * 副作用: 仅产生 5 分钟一次性 sms code 与登录会话,无业务数据落库,符合「验收造数不过夜」。
 */
@RunWith(AndroidJUnit4::class)
class RealBackendE2eTest {

    @Test
    fun loginWithRealSmsAndBrowseProducts() = runBlocking {
        Api.init(ApplicationProvider.getApplicationContext())
        val phone = "13900001234"

        UserApi.auth.smsCode(phone, "login")
        val code = runCatching { DebugApi.latestSmsCode(phone, "login").optString("code") }
            .getOrDefault("")
        assumeTrue("102 dev 模式未开或真码获取失败,跳过真实链路用例", code.isNotBlank())

        val tk = UserApi.auth.login(phone, "sms", code).optString("token")
        assertTrue("登录应返回 token", tk.isNotBlank())
        Api.setToken(tk)

        val items = ProductApi.list("broadband").optJSONArray("items")
        assertTrue("宽带产品列表应非空", items != null && items.length() > 0)

        // 实名读路径(6 条关键路径之一):测试账号已实名(唯一事实源),状态应为 VERIFIED
        val status = AccountApi.verifyStatus().optString("status")
        assertTrue("实名状态应为 VERIFIED,实为 $status", status == "VERIFIED")

        // 订单读路径:账号有历史真实订单(walkthrough 观察 ORD-2026082x-0003xx),列表应收敛非空
        val orders = OrderApi.list().optJSONArray("items")
        assertTrue("订单列表应非空(账号有历史订单)", orders != null && orders.length() > 0)

        // 账单读路径:信封契约(不抛异常即 code=0),响应应含契约键 currentDue/items
        val bills = BillApi.bills()
        assertTrue("账单响应应含契约键 currentDue", bills.has("currentDue"))
        assertTrue("账单响应应含契约键 items", bills.has("items"))
    }
}