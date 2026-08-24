package com.ymm.boss.user.page

import com.ymm.boss.user.ui.Route
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant

class MessageCategoryTest {

    @Test
    fun `categoryLabel maps known keys and falls back to 通知`() {
        assertEquals("账单缴费", categoryLabel("billing"))
        assertEquals("余额预警", categoryLabel("balance"))
        assertEquals("故障公告", categoryLabel("fault"))
        assertEquals("优惠活动", categoryLabel("promo"))
        assertEquals("通知", categoryLabel("unknown"))
        assertEquals("通知", categoryLabel(""))
    }

    @Test
    fun `routeOf routes each category to its landing page`() {
        assertEquals(Route.Bills, routeOf("billing"))
        assertEquals(Route.Topup, routeOf("balance"))
        assertEquals(Route.Fault, routeOf("fault"))
        assertEquals(Route.Coupon, routeOf("promo"))
        assertEquals(Route.Bills, routeOf("other"))
    }

    @Test
    fun `formatTime renders relative buckets`() {
        val now = Instant.now()
        assertTrue(formatTime(now.minusSeconds(30).toString()).endsWith("刚刚"))
        assertEquals("5分钟前", formatTime(now.minusSeconds(5 * 60).toString()))
        assertEquals("3小时前", formatTime(now.minusSeconds(3 * 3600).toString()))
        assertEquals("2天前", formatTime(now.minusSeconds(2 * 86400).toString()))
    }

    @Test
    fun `formatTime older than a week falls back to MM-dd`() {
        val raw = Instant.now().minusSeconds(30L * 86400).toString()
        val out = formatTime(raw)
        assertEquals(5, out.length)
        assertTrue(out.contains("-"))
    }

    @Test
    fun `formatTime blank or unparsable returns raw`() {
        assertEquals("", formatTime(""))
        assertEquals("not-a-time", formatTime("not-a-time"))
    }
}
