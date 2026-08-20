package com.ymm.boss.user.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import com.ymm.boss.user.ui.theme.ActionOrange
import com.ymm.boss.user.ui.theme.ActionOrangeDark
import com.ymm.boss.user.ui.theme.ActionPurple
import com.ymm.boss.user.ui.theme.ActionPurpleDark
import com.ymm.boss.user.ui.theme.BrandBlue
import com.ymm.boss.user.ui.theme.BrandBlueDark
import com.ymm.boss.user.ui.theme.BrandBlueGradientEnd
import com.ymm.boss.user.ui.theme.BrandBlueGradientEndDark
import com.ymm.boss.user.ui.theme.BrandBlueGradientStart
import com.ymm.boss.user.ui.theme.BrandBlueGradientStartDark
import com.ymm.boss.user.ui.theme.Green500
import com.ymm.boss.user.ui.theme.Green500Dark
import com.ymm.boss.user.ui.theme.OnSurfaceDark
import com.ymm.boss.user.ui.theme.OnSurfaceLight
import com.ymm.boss.user.ui.theme.OnSurfaceVariantDark
import com.ymm.boss.user.ui.theme.OnSurfaceVariantLight
import com.ymm.boss.user.ui.theme.SurfaceDark
import com.ymm.boss.user.ui.theme.SurfaceLight

object Palette {
    val primary = BrandBlue
    val primary2 = BrandBlueGradientEnd
    val bg = Color(0xFFF5F6F8)
    val panel = SurfaceLight
    val line = Color(0xFFE8EAED)
    val ink = OnSurfaceLight
    val muted = OnSurfaceVariantLight
    val subtle = Color(0xFFB0B3B8)
    val success = Green500
    val warn = ActionOrange
    val err = Color(0xFFFF3B30)
    val orange = ActionOrange
    val purple = ActionPurple
    val dotOn = Green500
    val dotOff = Color(0xFFFF3B30)
}

@Composable
fun BossTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val scheme = if (dark) {
        darkColorScheme(
            primary = BrandBlueDark,
            secondary = Green500Dark,
            tertiary = OnSurfaceVariantDark,
            background = Color(0xFF101114),
            surface = SurfaceDark,
            surfaceVariant = Color(0xFF2C2C2E),
            onBackground = OnSurfaceDark,
            onSurface = OnSurfaceDark,
            onSurfaceVariant = OnSurfaceVariantDark,
            error = Color(0xFFFF6961),
        )
    } else {
        lightColorScheme(
            primary = BrandBlue,
            secondary = Green500,
            tertiary = OnSurfaceVariantLight,
            background = Color(0xFFF5F6F8),
            surface = SurfaceLight,
            surfaceVariant = Color(0xFFE5E5EA),
            onBackground = OnSurfaceLight,
            onSurface = OnSurfaceLight,
            onSurfaceVariant = OnSurfaceVariantLight,
            error = Color(0xFFFF3B30),
        )
    }
    MaterialTheme(colorScheme = scheme, content = content)
}

@Composable
fun brandBlue(): Color = MaterialTheme.colorScheme.primary

@Composable
fun successGreen(): Color = MaterialTheme.colorScheme.secondary

@Composable
fun auxText(): Color = MaterialTheme.colorScheme.onSurfaceVariant

/** 统一渐变基准(与"我的"页一致):BrandBlue → 渐变端色。 */
@Composable
fun profileHeaderGradient(): Brush = if (isSystemInDarkTheme()) {
    Brush.linearGradient(listOf(BrandBlueDark, BrandBlueGradientEndDark))
} else {
    Brush.linearGradient(listOf(BrandBlue, BrandBlueGradientEnd))
}

/** 首页渐变:基准上仅起点略加深,保持同族不突兀。 */
@Composable
fun homeHeaderGradient(): Brush = if (isSystemInDarkTheme()) {
    Brush.linearGradient(listOf(Color(0xFF4E97EC), BrandBlueGradientEndDark))
} else {
    Brush.linearGradient(listOf(Color(0xFF006AE5), BrandBlueGradientEnd))
}

@Composable
fun actionOrange(): Color = if (isSystemInDarkTheme()) ActionOrangeDark else ActionOrange

@Composable
fun actionPurple(): Color = if (isSystemInDarkTheme()) ActionPurpleDark else ActionPurple
