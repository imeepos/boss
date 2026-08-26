package com.ymm.boss.user.api

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import kotlinx.coroutines.tasks.await

/**
 * 系统级定位薄封装。零第三方 Key，依赖 play-services-location。
 * 不做逆地理（后端无接口），拿到 WGS84 经纬度后由页面侧渲染为可读字符串。
 */
object LocationProvider {

    data class LatLng(val latitude: Double, val longitude: Double)

    sealed class Failure(message: String) : Exception(message) {
        object PermissionDenied : Failure("未授予定位权限")
        object Unavailable : Failure("未开启定位服务")
        object Timeout : Failure("定位超时，请重试或手动填写")
    }

    fun hasPermission(context: Context): Boolean {
        val fine = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION)
        val coarse = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION)
        return fine == PackageManager.PERMISSION_GRANTED || coarse == PackageManager.PERMISSION_GRANTED
    }

    @SuppressLint("MissingPermission")
    suspend fun current(context: Context): LatLng {
        if (!hasPermission(context)) throw Failure.PermissionDenied
        val client = LocationServices.getFusedLocationProviderClient(context.applicationContext)
        val token = CancellationTokenSource()
        try {
            val location: Location? = try {
                client.getCurrentLocation(Priority.PRIORITY_BALANCED_POWER_ACCURACY, token.token).await()
            } catch (_: Exception) { null }
                ?: try {
                    client.lastLocation.await()
                } catch (_: Exception) { null }
            return location?.let { LatLng(it.latitude, it.longitude) }
                ?: throw Failure.Unavailable
        } catch (e: Failure) {
            throw e
        } catch (_: Exception) {
            throw Failure.Unavailable
        } finally {
            token.cancel()
        }
    }

    /** 4 位小数 ≈ 11m，足够"参考位置"展示。 */
    fun format(p: LatLng): String =
        "%.4f°N, %.4f°E".format(p.latitude, p.longitude)
}