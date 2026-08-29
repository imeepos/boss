package com.ymm.boss.worker.ui.theme

import androidx.compose.ui.graphics.Color

// 对齐 docs/worker/style.css 的 :root 色板
val Primary = Color(0xFF1677FF)
val Primary2 = Color(0xFF69B1FF)
// 固定状态栏色:所有页面统一,取头部渐变起点色(对齐 user 端 statusBarSolid 方案)
val StatusBarSolid = Primary
// StatusBarSolid 的 ARGB 形式,供 SystemBarStyle(需要 Int)使用
const val StatusBarSolidArgb = 0xFF1677FF.toInt()
val Bg = Color(0xFFF5F6F8)
// Bg 的 ARGB 形式,供 SystemBarStyle(需要 Int 而非 Color)使用
const val BgArgb = 0xFFF5F6F8.toInt()
val Panel = Color(0xFFFFFFFF)
val Line = Color(0xFFE8EAED)
val Ink = Color(0xFF1F2329)
val Muted = Color(0xFF8C8C8C)
val Success = Color(0xFF52C41A)
val Warn = Color(0xFFFAAD14)
val Err = Color(0xFFFF4D4F)

// 标签色(tag-green/blue/orange/red/gray/cyan 前景+底色)
data class TagColor(val fg: Color, val bg: Color)

val TagGreen = TagColor(Color(0xFF389E0D), Color(0xFFF6FFED))
val TagBlue = TagColor(Color(0xFF1677FF), Color(0xFFE6F4FF))
val TagOrange = TagColor(Color(0xFFD46B08), Color(0xFFFFF7E6))
val TagRed = TagColor(Color(0xFFCF1322), Color(0xFFFFF1F0))
val TagGray = TagColor(Color(0xFF595959), Color(0xFFFAFAFA))
val TagCyan = TagColor(Color(0xFF08979C), Color(0xFFE6FFFB))

// QuadCell 状态浅色边框(状态色的弱化描边,TicketDetailCards 存量值成对收编)
val SuccessBorder = Color(0xFF6FD18B)
val ErrBorder = Color(0xFFFF9B9D)

// StatusLine 状态灯(蓝底渐变头部上的在线/离线小圆点,Widgets.kt 存量值成对收编)
val DotOnline = Color(0xFFA6E9A0)
val DotOffline = Color(0xFFFFA39E)

fun tagColor(status: String?): TagColor = when (status) {
    "DONE" -> TagGreen
    "ACCEPTED", "TODO" -> TagOrange
    "DOING" -> TagRed
    "SCAN_PENDING" -> TagBlue
    else -> TagBlue
}
