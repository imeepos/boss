package com.ymm.boss.worker.api

import org.json.JSONObject

object LocationApi {
    suspend fun report(lat: Double, lng: Double, accuracyM: Float, speedMps: Float, bearing: Float): JSONObject =
        Api.post("/location/report", JSONObject()
            .put("lat", lat)
            .put("lng", lng)
            .put("accuracyM", accuracyM)
            .put("speedMps", speedMps)
            .put("bearing", bearing)
            .put("timestampMs", System.currentTimeMillis()))
}
