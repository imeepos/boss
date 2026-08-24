package com.ymm.boss.worker.api

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

// UpdateApi.parse 升级判定投影(与用户端同构,契约 fields.md 8F)。
class UpdateApiParseTest {

    @Test
    fun parseUpdateAvailable() {
        val d = UpdateApi.parse(
            JSONObject(
                """{"updateAvailable":true,"force":false,"version":"1.1.0",
                "notes":"性能优化","sha256":"bb","size":52428800,
                "downloadUrl":"/client/apk/9"}"""
            )
        )
        assertTrue(d.updateAvailable)
        assertFalse(d.force)
        assertEquals("50.0MB", d.sizeMb)
        assertEquals("/client/apk/9", d.downloadUrl)
    }

    @Test
    fun parseUpToDateAndForce() {
        assertFalse(UpdateApi.parse(JSONObject("{}")).updateAvailable)
        assertTrue(UpdateApi.parse(JSONObject("""{"updateAvailable":true,"force":true}""")).force)
    }

    @Test
    fun sizeFormat() {
        assertEquals("2KB", UpdateApi.formatSize(2048))
        assertEquals("1.5MB", UpdateApi.formatSize((1.5 * 1048576).toLong()))
    }
}
