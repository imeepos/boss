package com.ymm.boss.user.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import com.ymm.boss.user.ui.theme.AuxGray
import com.ymm.boss.user.ui.theme.BrandBlue700
import com.ymm.boss.user.ui.theme.BrandBlue700Dark
import com.ymm.boss.user.ui.theme.SuccessGreen
import com.ymm.boss.user.ui.theme.SurfaceDark

/** 色板对齐 docs/user/style.css 的 :root 令牌。 */
object Palette {
    val primary = Color(0xFF1677FF)
    val primary2 = Color(0xFF69B1FF)
    val bg = Color(0xFFF5F6F8)
    val panel = Color(0xFFFFFFFF)
    val line = Color(0xFFE8EAED)
    val ink = Color(0xFF1C1C1C)
    val muted = Color(0xFF8C8C8C)
    val subtle = Color(0xFFB0B3B8)
    val success = Color(0xFF52C41A)
    val warn = Color(0xFFFAAD14)
    val err = Color(0xFFFF4D4F)
    val orange = Color(0xFFFA8C16)
    val purple = Color(0xFF722ED1)
    val dotOn = Color(0xFFA6E9A0)
    val dotOff = Color(0xFFFFA39E)
}

@Composable
fun BossTheme(content: @Composable () -> Unit) {
    val scheme = if (isSystemInDarkTheme()) {
        darkColorScheme().copy( // 设计稿只画浅色,深色按 F 节成对色值推演
            primary = SuccessGreen, tertiary = AuxGray,
            background = Color(0xFF1C1F26), surface = SurfaceDark,
            onBackground = Color(0xFFE6E8EC), onSurface = Color(0xFFE6E8EC),
        )
    } else {
        lightColorScheme(
            primary = SuccessGreen, tertiary = AuxGray,
            background = Palette.bg, surface = Palette.panel,
            onBackground = Palette.ink, onSurface = Palette.ink,
            error = Palette.err,
        )
    }
    MaterialTheme(colorScheme = scheme, content = content)
}

/** 主色 BrandBlue700 的主题感知取值:浅色用深蓝、深色用提亮蓝,双主题均可读。 */
@Composable
fun brandBlue(): Color = if (isSystemInDarkTheme()) BrandBlue700Dark else BrandBlue700
