package com.ymm.boss.user.page

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Circle
import androidx.compose.material.icons.outlined.Build
import androidx.compose.material.icons.outlined.LocalOffer
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Receipt
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import org.json.JSONObject
import java.time.Duration
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// 消息分类与样式工具:与 USER-APP-SPEC §2.2 IconTile 语义色规则同族。
internal data class CategoryUi(
    val key: String,
    val label: String,
    val icon: ImageVector?,
    val iconTint: Color,
)

internal val CATEGORIES = listOf(
    CategoryUi("", "全部", Icons.Filled.Circle, Palette.primary),
    CategoryUi("billing", "账单缴费", Icons.Filled.Circle, Palette.primary),
    CategoryUi("balance", "余额预警", Icons.Filled.Circle, Palette.orange),
    CategoryUi("fault", "故障公告", Icons.Filled.Circle, Palette.purple),
    CategoryUi("promo", "优惠活动", Icons.Filled.Circle, Palette.err),
)

internal fun routeOf(category: String): Route = when (category) {
    "balance" -> Route.Topup
    "fault" -> Route.Fault
    "promo" -> Route.Coupon
    "billing" -> Route.Bills
    else -> Route.Bills
}

internal fun colorOfCategory(category: String): Color = when (category) {
    "billing" -> Palette.primary
    "balance" -> Palette.orange
    "fault" -> Palette.purple
    "promo" -> Palette.err
    else -> Palette.muted
}

internal fun iconOfCategory(category: String): ImageVector = when (category) {
    "billing" -> Icons.Outlined.Receipt
    "balance" -> Icons.Outlined.Notifications
    "fault" -> Icons.Outlined.Build
    "promo" -> Icons.Outlined.LocalOffer
    else -> Icons.Outlined.Notifications
}

internal fun colorOfTag(tagLevel: String): Color = when (tagLevel) {
    "balance", "bill", "billing" -> Palette.orange
    "promo" -> Palette.err
    "fault" -> Palette.purple
    "info" -> Palette.primary
    else -> Palette.muted
}

internal fun categoryLabel(category: String): String = when (category) {
    "billing" -> "账单缴费"
    "balance" -> "余额预警"
    "fault" -> "故障公告"
    "promo" -> "优惠活动"
    else -> "通知"
}

// ISO8601 → 相对时间;解析失败原样返回。
internal fun formatTime(raw: String): String {
    if (raw.isBlank()) return ""
    return try {
        val instant = Instant.parse(raw)
        val mins = Duration.between(instant, Instant.now()).toMinutes()
        when {
            mins < 1 -> "刚刚"
            mins < 60 -> "${mins}分钟前"
            mins < 60 * 24 -> "${mins / 60}小时前"
            mins < 60 * 24 * 7 -> "${mins / (60 * 24)}天前"
            else -> LocalDateTime.ofInstant(instant, ZoneId.systemDefault())
                .format(DateTimeFormatter.ofPattern("MM-dd"))
        }
    } catch (e: Exception) { raw }
}