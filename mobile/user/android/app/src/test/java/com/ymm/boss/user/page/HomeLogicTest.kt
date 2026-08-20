package com.ymm.boss.user.page

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class HomeLogicTest {

    @Test
    fun `greeting covers all day segments`() {
        assertEquals("早上好", greetingFor(8))
        assertEquals("中午好", greetingFor(12))
        assertEquals("下午好", greetingFor(15))
        assertEquals("晚上好", greetingFor(22))
        assertEquals("晚上好", greetingFor(3))
    }

    @Test
    fun `status labels align with terms section 3`() {
        assertEquals("待核查", statusLabelOf("PENDING"))
        assertEquals("已预占", statusLabelOf("RESERVED"))
        assertEquals("装维中", statusLabelOf("INSTALLING"))
        assertEquals("已完成", statusLabelOf("DONE"))
        assertEquals("已取消", statusLabelOf("CANCELLED"))
        assertEquals("UNKNOWN", statusLabelOf("UNKNOWN"))
    }

    @Test
    fun `parseHome reads real envelope data payload`() {
        // 形状取自 2026-08-19 curl http://192.168.0.102:28080/api/user/v1/home
        val data = JSONObject(
            """
            {"customerName":"采购经理·王","phoneMasked":"139****1234",
             "onlineStatus":"服务在线 · 网络正常","hasUnread":false,
             "ongoingOrders":[{"orderNo":"ORD-20260818001","status":"INSTALLING",
               "statusLabel":"装维中","productName":"家庭宽带 1000M",
               "address":"广东省深圳市南山区科技园","stage":8}],
             "services":[]}
            """.trimIndent(),
        )
        val s = parseHome(data)
        assertEquals("采购经理·王", s.customerName)
        assertEquals("139****1234", s.phoneMasked)
        assertEquals(1, s.orders.size)
        assertEquals("ORD-20260818001", s.orders[0].orderNo)
        assertEquals(8, s.orders[0].stage)
        assertTrue(s.services.isEmpty())
        assertTrue(!s.loading)
    }

    @Test
    fun `parseHome tolerates missing arrays and blank statusLabel`() {
        val s = parseHome(
            JSONObject(
                """{"customerName":"王","phoneMasked":"139****1234",
                   "ongoingOrders":[{"orderNo":"ORD-1","status":"RESERVED","stage":3}]}""",
            ),
        )
        assertTrue(s.services.isEmpty())
        assertEquals("服务在线", s.onlineStatus)
        assertEquals("已预占", s.orders[0].statusLabel)
    }
}
