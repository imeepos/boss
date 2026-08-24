package com.ymm.boss.user.api

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

// UpdateApi.parse 升级判定投影(fields.md 8F 响应字段)。
class UpdateApiParseTest {

    @Test
    fun parseUpdateAvailable() {
        val d = UpdateApi.parse(
            JSONObject(
                """{"updateAvailable":true,"force":false,"version":"1.1.0",
                "versionCode":11,"notes":"修复已知问题","sha256":"aa","size":15728640,
                "downloadUrl":"/client/apk/3"}"""
            )
        )
        assertTrue(d.updateAvailable)
        assertFalse(d.force)
        assertEquals("1.1.0", d.version)
        assertEquals("修复已知问题", d.notes)
        assertEquals("15.0MB", d.sizeMb)
        assertEquals("/client/apk/3", d.downloadUrl)
    }

    @Test
    fun parseUpToDate() {
        val d = UpdateApi.parse(JSONObject("""{"updateAvailable":false}"""))
        assertFalse(d.updateAvailable)
        assertFalse(d.force)
        assertEquals("", d.downloadUrl)
    }

    @Test
    fun parseForce() {
        val d = UpdateApi.parse(JSONObject("""{"updateAvailable":true,"force":true}"""))
        assertTrue(d.force)
    }

    @Test
    fun sizeFormat() {
        assertEquals("", UpdateApi.formatSize(0))
        assertEquals("1KB", UpdateApi.formatSize(1024))
        assertEquals("1.0MB", UpdateApi.formatSize(1048576))
    }
}
