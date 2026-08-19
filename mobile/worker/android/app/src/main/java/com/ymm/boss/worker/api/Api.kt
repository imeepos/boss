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

object Api {
    var base: String = BuildConfig.BOSS_BASE_URL
    private const val TOKEN_KEY = "boss_worker_token"
    private const val PREFS = "boss_worker"
    private lateinit var appContext: Context

    fun init(ctx: Context) { appContext = ctx.applicationContext }

    fun token(): String = prefs().getString(TOKEN_KEY, "") ?: ""

    fun setToken(t: String?) {
        val e = prefs().edit()
        if (t.isNullOrEmpty()) e.remove(TOKEN_KEY) else e.putString(TOKEN_KEY, t)
        e.apply()
    }

    private fun prefs() = appContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    suspend fun get(path: String): JSONObject = JSONObject(request("GET", path, null))

    suspend fun getArray(path: String): JSONArray = JSONArray(request("GET", path, null))

    suspend fun post(path: String, body: JSONObject = JSONObject()): JSONObject =
        JSONObject(request("POST", path, body))

    suspend fun put(path: String, body: JSONObject = JSONObject()): JSONObject =
        JSONObject(request("PUT", path, body))

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

    private suspend fun request(method: String, path: String, body: JSONObject?): String =
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
                text
            } finally { conn.disconnect() }
        }
}
