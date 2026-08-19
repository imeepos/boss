package com.ymm.boss.user.api

import android.content.Context
import android.content.SharedPreferences
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

/**
 * 用户端统一接口层,对齐 docs/user/api.js。
 * 契约: api/openapi/user.yaml,前缀 /api/user/v1(真实服务端,mock 已移除)。
 * 模拟器访问宿主机服务用 10.0.2.2;真机可 adb reverse tcp:28080 tcp:28080 后改 127.0.0.1。
 */
object Api {
    const val DEFAULT_BASE = "http://10.0.2.2:28080/api/user/v1"
    private const val TOKEN_KEY = "boss_user_token"

    var base: String = DEFAULT_BASE
        private set

    private lateinit var prefs: SharedPreferences

    fun init(context: Context, baseOverride: String? = null) {
        prefs = context.getSharedPreferences("boss_user", Context.MODE_PRIVATE)
        if (!baseOverride.isNullOrBlank()) base = baseOverride
    }

    fun token(): String = if (::prefs.isInitialized) prefs.getString(TOKEN_KEY, "") ?: "" else ""

    fun setToken(t: String?) {
        if (!::prefs.isInitialized) return
        prefs.edit().apply { if (t.isNullOrBlank()) remove(TOKEN_KEY) else putString(TOKEN_KEY, t) }.apply()
    }

    class HttpError(val status: Int, message: String) : Exception(message)

    suspend fun get(path: String): JSONObject = request("GET", path, null)
    suspend fun getArray(path: String): JSONArray = requestArray("GET", path, null)
    suspend fun post(path: String, body: JSONObject? = JSONObject()): JSONObject = request("POST", path, body ?: JSONObject())
    suspend fun put(path: String, body: JSONObject): JSONObject = request("PUT", path, body)

    private suspend fun request(method: String, path: String, body: JSONObject?): JSONObject =
        withContext(Dispatchers.IO) {
            val conn = open(method, path, body)
            try {
                val code = conn.responseCode
                val text = streamText(conn, code)
                if (code !in 200..299) throw HttpError(code, "HTTP $code")
                if (text.isBlank()) JSONObject() else JSONObject(text)
            } finally { conn.disconnect() }
        }

    private suspend fun requestArray(method: String, path: String, body: JSONObject?): JSONArray =
        withContext(Dispatchers.IO) {
            val conn = open(method, path, body)
            try {
                val code = conn.responseCode
                val text = streamText(conn, code)
                if (code !in 200..299) throw HttpError(code, "HTTP $code")
                if (text.isBlank()) JSONArray() else JSONArray(text)
            } finally { conn.disconnect() }
        }

    private fun open(method: String, path: String, body: JSONObject?): HttpURLConnection {
        val conn = URL(base + path).openConnection() as HttpURLConnection
        conn.requestMethod = method
        conn.connectTimeout = 8000
        conn.readTimeout = 8000
        conn.setRequestProperty("Content-Type", "application/json")
        token().takeIf { it.isNotEmpty() }?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
        if (body != null) conn.outputStream.use { it.write(body.toString().toByteArray()) }
        return conn
    }

    private fun streamText(conn: HttpURLConnection, code: Int): String {
        val s = if (code in 200..299) conn.inputStream else conn.errorStream
        return s?.bufferedReader()?.use { it.readText() } ?: ""
    }

    /** URL query 拼接(跳过空值),等价 docs/user/api.js 的内联 qs。 */
    fun qs(params: Map<String, String?>): String {
        val sb = StringBuilder()
        params.forEach { (k, v) ->
            if (!v.isNullOrBlank()) {
                if (sb.isNotEmpty()) sb.append('&')
                sb.append(k).append('=').append(URLEncoder.encode(v, "UTF-8"))
            }
        }
        return if (sb.isEmpty()) "" else "?$sb"
    }
}
