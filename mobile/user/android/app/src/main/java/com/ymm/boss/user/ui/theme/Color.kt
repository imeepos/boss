package com.ymm.boss.user.ui.theme

import androidx.compose.ui.graphics.Color

/**
 * 设计稿 F 节色板,浅色/深色成对。
 * 页面内禁止硬编码 Color(0xFF...),新增色值一律先落这里并配深色值。
 */
val BrandBlue700 = Color(0xFF1E88E5) // 主色(浅色)
val BrandBlue700Dark = Color(0xFF90CAF9) // 主色(深色,提亮保证暗底可读)
val SurfaceDark = Color(0xFF1E1E1E) // 表面(深色)
val SuccessGreen = Color(0xFF4CAF50) // 成功/在网(浅色)
val SuccessGreenDark = Color(0xFF81C784) // 成功/在网(深色)
val AuxGray = Color(0xFF757575) // 辅助文字(浅色)
val AuxGrayDark = Color(0xFFBDBDBD) // 辅助文字(深色)
val GradientBlueStart = Color(0xFF1E88E5) // 渐变起点(浅色)
val GradientBlueStartDark = Color(0xFF0D47A1) // 渐变起点(深色)
val GradientBlueEnd = Color(0xFF42A5F5) // 渐变终点(浅色)
val GradientBlueEndDark = Color(0xFF1E3A5F) // 渐变终点(深色)
val OnGradient = Color(0xFFFFFFFF) // 渐变上的文字/图标(双主题同值,深蓝底白字均可读)
