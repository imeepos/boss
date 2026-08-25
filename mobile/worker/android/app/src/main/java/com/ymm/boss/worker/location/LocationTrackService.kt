package com.ymm.boss.worker.location

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.content.pm.PackageManager
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.LocationApi
import com.ymm.boss.worker.api.Api
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.cancel
import java.util.concurrent.TimeUnit

class LocationTrackService : Service() {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val client by lazy { LocationServices.getFusedLocationProviderClient(this) }
    private var lastReportedAt = 0L
    private var lastLat = Double.NaN
    private var lastLng = Double.NaN

    private val callback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            result.lastLocation?.let { location ->
                if (shouldReport(location.latitude, location.longitude)) {
                    lastReportedAt = System.currentTimeMillis()
                    lastLat = location.latitude
                    lastLng = location.longitude
                    scope.launch {
                        runCatching {
                            LocationApi.report(location.latitude, location.longitude, location.accuracy,
                                location.speed, location.bearing)
                        }
                    }
                }
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        startForeground(NOTIFICATION_ID, notification())
        requestUpdates()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (Api.token().isEmpty()) stopSelf()
        return START_STICKY
    }

    override fun onDestroy() {
        client.removeLocationUpdates(callback)
        scope.coroutineContext.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun requestUpdates() {
        val granted = ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED ||
            ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
        if (!granted) return
        val request = LocationRequest.Builder(Priority.PRIORITY_BALANCED_POWER_ACCURACY, REPORT_INTERVAL_MS)
            .setMinUpdateIntervalMillis(REPORT_INTERVAL_MS)
            .setWaitForAccurateLocation(false)
            .build()
        client.requestLocationUpdates(request, callback, mainLooper)
    }

    private fun shouldReport(lat: Double, lng: Double): Boolean {
        if (lastReportedAt == 0L) return true
        if (System.currentTimeMillis() - lastReportedAt >= TimeUnit.MINUTES.toMillis(5)) return true
        return kotlin.math.abs(lat - lastLat) >= 0.0001 || kotlin.math.abs(lng - lastLng) >= 0.0001
    }

    private fun notification(): Notification = NotificationCompat.Builder(this, CHANNEL_ID)
        .setContentTitle(getString(R.string.app_name))
        .setContentText("正在上报实时位置")
        .setSmallIcon(android.R.drawable.ic_menu_mylocation)
        .setOngoing(true)
        .setPriority(NotificationCompat.PRIORITY_LOW)
        .build()

    companion object {
        private const val CHANNEL_ID = "location_tracking"
        private const val NOTIFICATION_ID = 1201
        private const val REPORT_INTERVAL_MS = 60_000L

        fun createChannel(manager: NotificationManager) {
            manager.createNotificationChannel(NotificationChannel(CHANNEL_ID, "实时位置", NotificationManager.IMPORTANCE_LOW))
        }
    }
}
