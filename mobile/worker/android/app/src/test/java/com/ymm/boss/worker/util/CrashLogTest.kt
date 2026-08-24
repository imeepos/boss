package com.ymm.boss.worker.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class CrashLogTest {
    @Test
    fun format_contains_thread_and_stack() {
        val out = CrashLog.format(Thread.currentThread(), IllegalStateException("boom"))
        assertTrue(out.contains("thread:"))
        assertTrue(out.contains("IllegalStateException: boom"))
    }

    @Test
    fun pruneOldest_keeps_newest_n() {
        val dir = File.createTempFile("crash", "").let { it.delete(); it.apply { mkdirs() } }
        try {
            val files = (1..5).map { i ->
                File(dir, "crash-20260828-00000$i.txt").apply { writeText("x") }
            }
            CrashLog.pruneOldest(files, 3)
            val left = dir.listFiles()!!.map { it.name }.sorted()
            assertEquals(listOf("crash-20260828-000003.txt", "crash-20260828-000004.txt", "crash-20260828-000005.txt"), left)
        } finally {
            dir.deleteRecursively()
        }
    }
}
