package com.ymm.boss.worker.util

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationServices
import com.google.android.gms.tasks.Tasks
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

data class LocationResult(val lat: Double, val lng: Double, val accuracy: Float)

object LocationHelper {
    suspend fun getCurrentLocation(ctx: Context): LocationResult? = withContext(Dispatchers.IO) {
        val fine = ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED
        val coarse = ContextCompat.checkSelfPermission(ctx, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
        if (!fine && !coarse) return@withContext null
        val client = LocationServices.getFusedLocationProviderClient(ctx)
        try {
            val loc: Location? = Tasks.await<Location?>(client.lastLocation)
            if (loc != null) return@withContext LocationResult(loc.latitude, loc.longitude, loc.accuracy)
            // 首次定位可能需要时间,返回 null 让 UI 提示
            null
        } catch (_: Exception) { null }
    }
}
