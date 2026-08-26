package com.ymm.boss.user.api

import org.junit.Assert.assertEquals
import org.junit.Test

/** LocationProvider.format 是纯函数(只依赖 Kotlin stdlib),在 JVM 单测里直接跑。 */
class LocationProviderFormatTest {

    @Test fun formatsBeijingAsFourDecimals() {
        val s = LocationProvider.format(LocationProvider.LatLng(39.90420, 116.40740))
        assertEquals("39.9042°N, 116.4074°E", s)
    }

    @Test fun zeroPointFormatsCleanly() {
        val s = LocationProvider.format(LocationProvider.LatLng(0.0, 0.0))
        assertEquals("0.0000°N, 0.0000°E", s)
    }

    @Test fun negativeLongitudeFormatsNegativeE() {
        val s = LocationProvider.format(LocationProvider.LatLng(33.6844, -117.8265))
        // Kotlin String.format 保留 -117.8265;首位 - 在数字前
        assertEquals("33.6844°N, -117.8265°E", s)
    }
}