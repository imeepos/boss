package com.ymm.boss.user.api

import android.content.Context
import com.ymm.boss.user.BuildConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

/**
 * 用户端统一接口层,对齐 docs/user/api.js。
 * 契约: api/openapi/user.yaml,前缀 /api/user/v1(真实服务端,mock 已移除)。
 * 联调统一连接 192.168.0.102:28080;模拟器/真机均直连此地址。
 */
object Api {
    const val DEFAULT_BASE = BuildConfig.BOSS_BASE_URL

    var base: String = DEFAULT_BASE
        private set

    var onUnauthorized: (() -> Unit)? = null

    fun init(context: Context, baseOverride: String? = null) {
        TokenStore.init(context)
        LangStore.init(context)
        if (!baseOverride.isNullOrBlank()) base = baseOverride
    }

    fun token(): String = TokenStore.read()

    fun setToken(t: String?) = TokenStore.write(t)

    private fun handleUnauthorized(status: Int) {
        if (isUnauthorized(status)) {
            setToken(null)
            onUnauthorized?.invoke()
        }
    }

    private fun isUnauthorized(status: Int): Boolean = status == 401 || status == 40100

    class HttpError(val status: Int, message: String) : Exception(message)

    /**
     * 异常 → 用户可读文案,对齐 pkg/apitypes/code.go 统一错误码。
     * 登录等场景可对 40100(凭证无效)按上下文二次细化(HttpError.status 即信封 code)。
     */
    fun friendlyMessage(e: Exception): String = when {
        e is HttpError && e.status >= 1000 -> when (e.status) {
            40100 -> "凭证无效，请重新操作"
            40300 -> "暂无权限，请联系客服"
            40400 -> "账号不存在，请先注册"
            40900, 40910, 40920 -> "操作冲突，请刷新后重试"
            42200 -> "输入格式不正确，请检查后重填"
            42300 -> "资源被占用，请稍后重试"
            50000 -> "服务开小差了，请稍后重试"
            50200 -> "服务暂不可用，请稍后重试"
            else -> e.message?.ifBlank { null } ?: "请求失败(${e.status})"
        }
        e is HttpError -> when (e.status) { // HTTP 层错误(status < 1000)
            401 -> "登录状态已失效，请重新登录"
            404 -> "接口不存在，请更新 App"
            in 500..599 -> "服务器异常，请稍后重试"
            else -> "网络异常(HTTP ${e.status})"
        }
        e is java.io.IOException -> "网络连接失败，请检查网络"
        else -> e.message?.ifBlank { null } ?: "操作失败，请重试"
    }

    suspend fun get(path: String): JSONObject = request("GET", path, null)
    suspend fun getArray(path: String): JSONArray = requestArray("GET", path, null)
    suspend fun post(path: String, body: JSONObject? = JSONObject()): JSONObject = request("POST", path, body ?: JSONObject())
    suspend fun put(path: String, body: JSONObject): JSONObject = request("PUT", path, body)

    /** multipart 单文件上传(带 Bearer,字段名 file),返回信封 data;非 2xx 抛 HttpError。 */
    suspend fun upload(path: String, fileName: String, contentType: String, bytes: ByteArray): JSONObject =
        withContext(Dispatchers.IO) {
            val boundary = "----boss${System.currentTimeMillis()}"
            val conn = URL(base + path).openConnection() as HttpURLConnection
            conn.requestMethod = "POST"
            conn.doOutput = true
            conn.connectTimeout = 15000
            conn.readTimeout = 30000
            conn.setRequestProperty("Content-Type", "multipart/form-data; boundary=$boundary")
            token().takeIf { it.isNotEmpty() }?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
            java.io.DataOutputStream(conn.outputStream).use { out ->
                out.writeBytes("--$boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"$fileName\"\r\n")
                out.writeBytes("Content-Type: $contentType\r\n\r\n")
                out.write(bytes)
                out.writeBytes("\r\n--$boundary--\r\n")
            }
            try {
                val code = conn.responseCode
                val text = streamText(conn, code)
                if (code !in 200..299) {
                    handleUnauthorized(code)
                    throw HttpError(code, "HTTP $code")
                }
                if (text.isBlank()) JSONObject() else unwrap(text)
            } finally { conn.disconnect() }
    }

    /** 认证下载二进制(带 Bearer 头),用于 PDF 凭证/发票,非 2xx 抛 HttpError。 */
    suspend fun getBytes(path: String): ByteArray = withContext(Dispatchers.IO) {
        val conn = open("GET", path, null)
        try {
            val code = conn.responseCode
            if (code !in 200..299) {
                handleUnauthorized(code)
                throw HttpError(code, "HTTP $code")
            }
            val buf = ByteArrayOutputStream()
            conn.inputStream.use { it.copyTo(buf) }
            buf.toByteArray()
        } finally { conn.disconnect() }
    }

    /** 解信封 {code,msg,data}:code!=0 抛 HttpError,成功返回 data(缺省空对象)。 */
    private fun unwrap(text: String): JSONObject {
        val obj = JSONObject(text)
        val code = obj.optInt("code", -1)
        if (code != 0) {
            handleUnauthorized(code)
            throw HttpError(code, obj.optString("msg").ifBlank { "code $code" })
        }
        return obj.optJSONObject("data") ?: JSONObject()
    }

    private suspend fun request(method: String, path: String, body: JSONObject?): JSONObject =
        withContext(Dispatchers.IO) {
            val conn = open(method, path, body)
            try {
                val code = conn.responseCode
                val text = streamText(conn, code)
                if (code !in 200..299) {
                    handleUnauthorized(code)
                    throw HttpError(code, "HTTP $code")
                }
                if (text.isBlank()) JSONObject() else unwrap(text)
            } finally { conn.disconnect() }
        }

    private suspend fun requestArray(method: String, path: String, body: JSONObject?): JSONArray =
        withContext(Dispatchers.IO) {
            val conn = open(method, path, body)
            try {
                val code = conn.responseCode
                val text = streamText(conn, code)
                if (code !in 200..299) {
                    handleUnauthorized(code)
                    throw HttpError(code, "HTTP $code")
                }
                if (text.isBlank()) return@withContext JSONArray()
                // 数组载荷包在信封 data 内:data 本身为数组,或 data.items
                val obj = JSONObject(text)
                val envelopeCode = obj.optInt("code", -1)
                if (envelopeCode != 0) {
                    handleUnauthorized(envelopeCode)
                    throw HttpError(envelopeCode, obj.optString("msg"))
                }
                when (val d = obj.opt("data")) {
                    is JSONArray -> d
                    is JSONObject -> d.optJSONArray("items") ?: JSONArray()
                    else -> JSONArray()
                }
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
