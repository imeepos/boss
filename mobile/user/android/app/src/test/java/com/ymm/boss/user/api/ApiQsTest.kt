package com.ymm.boss.user.api

import org.junit.Assert.assertEquals
import org.junit.Test

class ApiQsTest {

    @Test
    fun `qs skips null and blank values`() {
        val q = Api.qs(mapOf("status" to "DONE", "category" to null, "period" to ""))
        assertEquals("?status=DONE", q)
    }

    @Test
    fun `qs returns empty when all values empty`() {
        assertEquals("", Api.qs(mapOf("a" to null, "b" to "")))
    }

    @Test
    fun `qs url-encodes reserved characters`() {
        val q = Api.qs(mapOf("no" to "ORD 1/2"))
        assertEquals("?no=ORD+1%2F2", q)
    }
}
