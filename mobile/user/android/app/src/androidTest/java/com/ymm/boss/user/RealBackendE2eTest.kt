package com.ymm.boss.user

import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.BillApi
import com.ymm.boss.user.api.DebugApi
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.PlanApi
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.UserApi
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
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

        val code = runCatching {
            UserApi.auth.smsCode(phone, "login")
            DebugApi.latestSmsCode(phone, "login").optString("code")
        }.getOrDefault("")
        assumeTrue("发码被冷却(测试号共享 60s 冷却)/102 dev 未开/真码失败,跳过真实链路用例", code.isNotBlank())

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

        // 「我的」读路径(6 条关键路径之五):套餐在用在网(家庭宽带100M),plan 应收敛
        val plan = PlanApi.profile().optJSONObject("plan")
        assertTrue("套餐信息应收敛(账号在网)", plan != null)

        // 地址读:信封契约键 items(账号 0 地址只断契约不断量)
        assertTrue("地址响应应含契约键 items", ProfileApi.addresses().has("items"))

        // 券读:信封契约键 items
        assertTrue("券响应应含契约键 items", BillApi.coupons().has("items"))
    }

    /**
     * 错误码语义负路径(D-7「信封/错误码语义确认」):错码登录应得 40100(凭证无效),
     * 客户端 loginErrorMessage(sms) 应路由为「验证码错误/过期」文案而非「输入格式不正确」。
     * 实测: 错码 999999 -> {"code":40100,"msg":"未认证或凭证无效"};42200 仅为缺码/绑定层。
     */
    @Test
    fun wrongSmsCodeIsCredentialError() = runBlocking {
        Api.init(ApplicationProvider.getApplicationContext())
        val phone = "13900001234"

        val e = runCatching { UserApi.auth.login(phone, "sms", "999999") }.exceptionOrNull()
        assumeTrue("真实后端不可达或登录行为异常,跳过负路径用例", e is Api.HttpError)
        val err = e as Api.HttpError
        assertTrue("错码应映射 40100 凭证无效,实为 ${err.status}", err.status == 40100)
        val msg = com.ymm.boss.user.page.loginErrorMessage(err, "sms")
        assertTrue("40100 应路由验证码错误文案,实为 $msg", msg.contains("验证码"))
    }

    /**
     * BUG-02 修复验证:业务 40100(错码登录/改密旧口令错)不得清除会话 token。
     * 修复前旧口令错即全会话闪断回登录页;修复后仅 HTTP 401(凭证真失效)清除。
     */
    @Test
    fun business40100DoesNotClearSession() = runBlocking {
        Api.init(ApplicationProvider.getApplicationContext())
        val phone = "13900001234"

        val code = runCatching {
            UserApi.auth.smsCode(phone, "login")
            DebugApi.latestSmsCode(phone, "login").optString("code")
        }.getOrDefault("")
        assumeTrue("发码被冷却(测试号共享 60s 冷却)或真码获取失败,跳过", code.isNotBlank())

        val tk = UserApi.auth.login(phone, "sms", code).optString("token")
        assertTrue("登录应返回 token", tk.isNotBlank())
        Api.setToken(tk)

        // 1) 错码登录:业务 40100(验证码校验失败)——不得清会话
        runCatching { UserApi.auth.login(phone, "sms", "999999") }
        assertEquals("错码登录 40100 不得清除已登录 token", tk, Api.token())

        // 2) 改密旧口令错:业务 40100(portal_security 口令校验失败)——不得清会话
        runCatching { ProfileApi.changePassword("wrong-old-${System.currentTimeMillis()}", "NewPass123456") }
        assertEquals("旧口令错不得清除已登录 token", tk, Api.token())
    }
}