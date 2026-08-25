package com.ymm.boss.user.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import com.ymm.boss.user.ui.theme.ActionOrange
import com.ymm.boss.user.ui.theme.ActionPurple
import com.ymm.boss.user.ui.theme.BrandBlue
import com.ymm.boss.user.ui.theme.BrandBlueGradientEnd
import com.ymm.boss.user.ui.theme.Green500
import com.ymm.boss.user.ui.theme.OnSurfaceLight
import com.ymm.boss.user.ui.theme.OnSurfaceVariantLight
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

// 固定浅色主题,不跟随系统暗色模式(用户裁定 2026-08-25):手机开暗色时
// 卡片/底部导航曾切 SurfaceDark 变黑,其余页面写死浅色不变 → 主题撕裂。
// 与 worker 端 WorkerTheme 同构,暗色配色完全移除。
@Composable
fun BossTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = lightColorScheme(
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
        ),
        content = content,
    )
}

@Composable
fun brandBlue(): Color = MaterialTheme.colorScheme.primary

@Composable
fun successGreen(): Color = MaterialTheme.colorScheme.secondary

@Composable
fun auxText(): Color = MaterialTheme.colorScheme.onSurfaceVariant

/** 统一渐变基准(与"我的"页一致):BrandBlue → 渐变端色。 */
@Composable
fun profileHeaderGradient(): Brush =
    Brush.linearGradient(listOf(BrandBlue, BrandBlueGradientEnd))

/** 首页渐变:基准上仅起点略加深,保持同族不突兀。 */
@Composable
fun homeHeaderGradient(): Brush =
    Brush.linearGradient(listOf(Color(0xFF006AE5), BrandBlueGradientEnd))

/** 固定状态栏色:所有页面统一,取首页渐变起点色,不透明、不随页面切换变化。 */
@Composable
fun statusBarSolid(): Color = Color(0xFF006AE5)

@Composable
fun actionOrange(): Color = ActionOrange

@Composable
fun actionPurple(): Color = ActionPurple
