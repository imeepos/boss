package com.ymm.boss.worker.util

import android.content.Context
import com.ymm.boss.worker.api.Api
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.io.File
import java.io.PrintWriter
import java.io.StringWriter
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * 本地崩溃采集:默认 UncaughtExceptionHandler 前置挂钩,崩溃堆栈落
 * filesDir/crash/crash-<时间戳>.txt,保留最近 [KEEP] 份,写完链回原
 * handler(不吞系统崩溃对话框/进程退出)。上传端点待后端契约,先本地
 * 留痕,adb run-as 可直接拉取排查。
 */
object CrashLog {
    private const val DIR = "crash"
    private const val KEEP = 10

    fun install(appContext: Context) {
        val prev = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { t, e ->
            runCatching { write(appContext, t, e) }
            prev?.uncaughtException(t, e)
        }
    }

    /** 崩溃文件清单(旧→新),供上传/排查用。 */
    fun files(ctx: Context): List<File> =
        dir(ctx).listFiles { f -> f.isFile }?.sortedBy { it.name } ?: emptyList()

    /**
     * 启动补传:逐条 POST /client/crash(worker 契约),成功即删本地文件防重传;
     * 未登录静默跳过,单文件失败不阻塞后续。自身起 IO 协程,调用方可直接主线程调。
     */
    fun uploadPending(ctx: Context) {
        CoroutineScope(Dispatchers.IO).launch {
            if (Api.token().isEmpty()) return@launch
            for (f in files(ctx)) {
                try {
                    Api.post("/client/crash", JSONObject()
                        .put("app", "boss-worker/" + com.ymm.boss.worker.BuildConfig.VERSION_NAME)
                        .put("log", f.readText()))
                    runCatching { f.delete() }
                } catch (_: Exception) {
                    // 保留文件下次再传
                }
            }
        }
    }

    private fun write(ctx: Context, thread: Thread, e: Throwable) {
        val d = dir(ctx)
        d.mkdirs()
        val ts = SimpleDateFormat("yyyyMMdd-HHmmss-SSS", Locale.US).format(Date())
        File(d, "crash-$ts.txt").writeText(format(thread, e))
        pruneOldest(d.listFiles { f -> f.isFile }?.toList() ?: emptyList(), KEEP)
    }

    private fun dir(ctx: Context) = File(ctx.filesDir, DIR)

    internal fun format(thread: Thread, e: Throwable): String {
        val sw = StringWriter()
        e.printStackTrace(PrintWriter(sw))
        return buildString {
            appendLine("thread: ${thread.name}")
            appendLine("time: ${System.currentTimeMillis()}")
            appendLine("stack:")
            append(sw.toString())
        }
    }

    /** 只保留最新 keep 份(文件名字典序=时间序)。 */
    internal fun pruneOldest(files: List<File>, keep: Int) {
        val sorted = files.sortedBy { it.name }
        val excess = sorted.size - keep
        if (excess > 0) sorted.take(excess).forEach { runCatching { it.delete() } }
    }
}
