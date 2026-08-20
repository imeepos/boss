package com.ymm.boss.user.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/** 通用组件库,对应 docs/user/style.css 的 .card/.cell/.tag/.topbar/.notice。 */

@Composable
fun AppCard(
    modifier: Modifier = Modifier,
    shape: RoundedCornerShape = RoundedCornerShape(12.dp),
    outer: PaddingValues = PaddingValues(horizontal = 14.dp, vertical = 6.dp),
    content: @Composable () -> Unit,
) {
    // Column 而非 Box:全部调用点的语义都是卡片内纵向堆叠,Box 会让
    // 多个子元素叠在左上角(真机已两次踩中文字/按钮叠印)。
    Column(
        modifier
            .padding(outer)
            .background(Palette.panel, shape)
            .padding(16.dp)
    ) { content() }
}

@Composable
fun CardTitle(title: String, more: String? = null, onMore: (() -> Unit)? = null) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
        Text(title, fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink, modifier = Modifier.weight(1f))
        if (more != null) Text(more, fontSize = 12.5.sp, color = Palette.muted, modifier = Modifier.clickable { onMore?.invoke() })
    }
}

@Composable
fun CellRow(title: String, desc: String? = null, onClick: (() -> Unit)? = null, right: (@Composable () -> Unit)? = null) {
    Row(
        Modifier.fillMaxWidth().clickable(enabled = onClick != null) { onClick?.invoke() }.padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink)
            if (!desc.isNullOrBlank()) Text(desc, fontSize = 12.sp, color = Palette.muted)
        }
        right?.invoke()
    }
}

@Composable
fun Tag(text: String, color: Color = Palette.primary) {
    Text(
        text, fontSize = 11.sp, color = color,
        modifier = Modifier.background(color.copy(alpha = 0.1f), RoundedCornerShape(4.dp)).padding(horizontal = 6.dp, vertical = 2.dp),
    )
}

@Composable
fun TopBar(title: String, onBack: (() -> Unit)? = null, action: String? = null, onAction: (() -> Unit)? = null) {
    Row(
        Modifier.fillMaxWidth().height(48.dp)
            .background(Brush.linearGradient(listOf(Palette.primary, Palette.primary2)))
            .padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (onBack != null) Text("‹", color = Color.White, fontSize = 20.sp, modifier = Modifier.clickable { onBack() })
        Text(title, color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.W600, modifier = Modifier.weight(1f).padding(start = 8.dp))
        if (action != null) Text(action, color = Color.White, fontSize = 13.sp, modifier = Modifier.clickable { onAction?.invoke() })
    }
}

@Composable
fun StatusDot(on: Boolean = true) {
    Box(Modifier.size(8.dp).background(if (on) Palette.dotOn else Palette.dotOff, CircleShape))
}

@Composable
fun Notice(text: String, color: Color = Palette.muted) {
    Text(text, fontSize = 12.5.sp, color = color, modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp))
}

/** tab 页居中标题栏:白底 48dp,对应设计稿 products/orders 页头。 */
@Composable
fun TabHeader(title: String) {
    Box(
        Modifier.fillMaxWidth().height(48.dp).background(Palette.panel),
        contentAlignment = Alignment.Center,
    ) { Text(title, fontSize = 16.sp, fontWeight = FontWeight.W600, color = Palette.ink) }
}

/** 胶囊筛选 tab:选中实心主色,未选中白底描边(plain 时未选中无底色)。 */
@Composable
fun PillTab(label: String, active: Boolean, onClick: () -> Unit, icon: ImageVector? = null, plain: Boolean = false) {
    val bg = if (active) Palette.primary else if (plain) Color.Transparent else Palette.panel
    val borderColor = if (active) Palette.primary else if (plain) Color.Transparent else Palette.line
    Row(
        Modifier
            .background(bg, RoundedCornerShape(999.dp))
            .border(1.dp, borderColor, RoundedCornerShape(999.dp))
            .clickable { onClick() }
            .padding(horizontal = 14.dp, vertical = 7.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) {
            Icon(icon, contentDescription = null, tint = if (active) Color.White else Palette.muted, modifier = Modifier.size(15.dp))
            Spacer(Modifier.width(5.dp))
        }
        Text(
            label, fontSize = 13.sp,
            color = if (active) Color.White else if (plain) Palette.muted else Palette.ink,
            fontWeight = if (active) FontWeight.W500 else FontWeight.Normal,
        )
    }
}

/** 浅底圆角图标容器:tint 10% 底 + 同色图标,列表行/快捷入口通用。 */
@Composable
fun IconTile(icon: ImageVector, tint: Color, size: Dp = 40.dp, corner: Dp = 10.dp) {
    Box(
        Modifier.size(size).background(tint.copy(alpha = 0.1f), RoundedCornerShape(corner)),
        contentAlignment = Alignment.Center,
    ) { Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(size * 0.55f)) }
}

@Composable
fun FieldLabel(text: String) {
    Text(text, fontSize = 13.sp, color = Palette.muted, modifier = Modifier.padding(bottom = 4.dp))
}
