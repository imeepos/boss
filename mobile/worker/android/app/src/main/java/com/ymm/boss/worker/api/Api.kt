package com.ymm.boss.worker.api

import android.content.Context
import com.ymm.boss.worker.BuildConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

class ApiException(val status: Int, message: String) : Exception(message)

// 师傅端统一 HTTP 客户端,契约对齐 api/openapi/worker.yaml(信封裁定见 alignment-audit.md §9)。
// 服务端所有响应为统一信封 {code,msg,data}:code!=0 抛 ApiException,成功返回 data 载荷。
// base 由 BuildConfig.BOSS_BASE_URL 注入,debug 走 10.0.2.2:28080(模拟器宿主机),release 走 HTTPS 生产域名。
object Api {
    var base: String = BuildConfig.BOSS_BASE_URL
    private const val TOKEN_KEY = "boss_worker_token"
    private const val PREFS = "boss_worker"
    private lateinit var appContext: Context

    fun init(ctx: Context) {
        appContext = ctx.applicationContext
    }

    fun token(): String = prefs().getString(TOKEN_KEY, "") ?: ""

    fun setToken(t: String?) {
        val e = prefs().edit()
        if (t.isNullOrEmpty()) e.remove(TOKEN_KEY) else e.putString(TOKEN_KEY, t)
        e.apply()
    }

    private fun prefs() = appContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    suspend fun get(path: String): JSONObject = request("GET", path, null)

    suspend fun getArray(path: String): JSONArray = requestArray("GET", path)

    suspend fun post(path: String, body: JSONObject = JSONObject()): JSONObject =
        request("POST", path, body)

    suspend fun put(path: String, body: JSONObject = JSONObject()): JSONObject =
        request("PUT", path, body)

    suspend fun upload(path: String, fileName: String, contentType: String, bytes: ByteArray): JSONObject =
        withContext(Dispatchers.IO) {
            val boundary = "----boss-${System.currentTimeMillis()}"
            val conn = open("POST", path)
            conn.setRequestProperty("Content-Type", "multipart/form-data; boundary=$boundary")
            conn.doOutput = true
            try {
                conn.outputStream.use { out ->
                    out.write("--$boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"$fileName\"\r\nContent-Type: $contentType\r\n\r\n".toByteArray())
                    out.write(bytes)
                    out.write("\r\n--$boundary--\r\n".toByteArray())
                }
                val code = conn.responseCode
                val text = (if (code in 200..299) conn.inputStream else conn.errorStream)?.bufferedReader()?.readText() ?: ""
                if (code !in 200..299) throw ApiException(code, "HTTP $code")
                unwrap(text)
            } finally { conn.disconnect() }
        }

    private fun open(method: String, path: String): HttpURLConnection {
        val conn = URL(base + path).openConnection() as HttpURLConnection
        conn.requestMethod = method
        conn.connectTimeout = 10_000
        conn.readTimeout = 15_000
        conn.setRequestProperty("Content-Type", "application/json")
        val tk = token()
        if (tk.isNotEmpty()) conn.setRequestProperty("Authorization", "Bearer $tk")
        return conn
    }

    // 解信封:业务错误(HTTP 仍 200)转 ApiException,成功返回 data(缺省空对象)。
    private fun unwrap(text: String): JSONObject {
        val obj = JSONObject(text)
        val code = obj.optInt("code", -1)
        if (code != 0) throw ApiException(code, obj.optString("msg").ifBlank { "code $code" })
        return obj.optJSONObject("data") ?: JSONObject()
    }

    private suspend fun request(method: String, path: String, body: JSONObject?): JSONObject =
        withContext(Dispatchers.IO) {
            val conn = open(method, path)
            try {
                if (body != null) {
                    conn.doOutput = true
                    conn.outputStream.use { it.write(body.toString().toByteArray()) }
                }
                val code = conn.responseCode
                val text = (if (code in 200..299) conn.inputStream else conn.errorStream)
                    ?.bufferedReader()?.readText() ?: ""
                if (code !in 200..299) throw ApiException(code, "HTTP $code")
                if (text.isBlank()) JSONObject() else unwrap(text)
            } finally {
                conn.disconnect()
            }
        }

    // 数组载荷包在信封 data 内:data 本身为数组,或 data.items。
    private suspend fun requestArray(method: String, path: String): JSONArray =
        withContext(Dispatchers.IO) {
            val conn = open(method, path)
            try {
                val code = conn.responseCode
                val text = (if (code in 200..299) conn.inputStream else conn.errorStream)
                    ?.bufferedReader()?.readText() ?: ""
                if (code !in 200..299) throw ApiException(code, "HTTP $code")
                if (text.isBlank()) return@withContext JSONArray()
                val obj = JSONObject(text)
                val ec = obj.optInt("code", -1)
                if (ec != 0) throw ApiException(ec, obj.optString("msg").ifBlank { "code $ec" })
                when (val d = obj.opt("data")) {
                    is JSONArray -> d
                    is JSONObject -> d.optJSONArray("items") ?: JSONArray()
                    else -> JSONArray()
                }
            } finally {
                conn.disconnect()
            }
        }
}
