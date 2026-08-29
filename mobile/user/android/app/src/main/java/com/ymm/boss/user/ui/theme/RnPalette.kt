package com.ymm.boss.user.ui.theme

import androidx.compose.ui.graphics.Color

/** 登录/注册/找回/实名族专属色板（第二主题），designs/login-register-states-v2 背书 + 真机验收。
 *  「并入全局 vs 维持第二主题」设计拍板前冻结新增色值，族内仅允许引用本区常量。
 *  本文件已入 UI 门禁豁免白名单（scripts/check-ui-consistency.mjs user 端 exempt）。 */
object RnPalette {
    val primary = Color(0xFF086CF5)
    val heroStart = Color(0xFF0872F4) // 品牌渐变起点(Stepper/登录 Hero)
    val heroMid = Color(0xFF0B82F8) // 品牌渐变中点(登录 Hero 三段)
    val heroEnd = Color(0xFF1698FA)
    val success = Color(0xFF0AA847)
    val successBg = Color(0xFFEFFFF4)
    val warn = Color(0xFFF57900)
    val warnBg = Color(0xFFFFF5E8)
    val ink = Color(0xFF171B23)
    val muted = Color(0xFF5F6671)
    val placeholder = Color(0xFFAEB4BE)
    val line = Color(0xFFE7EAF0)
    val error = Color(0xFFFF2D2F) // 错误/失败提示红
    val pageBg = Color(0xFFF6F8FA) // 登录页底
    val fieldBg = Color(0xFFF7F8FA) // 输入行填充底
    val iconTint = Color(0xFF7D8593) // 输入行前缀图标
    val segmentBg = Color(0xFFF5F6F8) // 分段控件槽底
    val checkBorder = Color(0xFFD1D7E0) // 协议勾选未选中描边
}
