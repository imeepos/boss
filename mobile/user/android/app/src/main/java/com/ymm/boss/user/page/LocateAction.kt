package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.MyLocation
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Palette

/**
 * 地址弹层顶部的"使用当前位置"卡片。
 * locating=true 时禁用点击（防重入），hint 有值时边框变主色作为已获取过的提示。
 */
@Composable
internal fun LocateAction(locating: Boolean, hint: String, onClick: () -> Unit) {
    val border = if (hint.isNotBlank()) Palette.primary else Palette.line
    Row(
        Modifier.fillMaxWidth()
            .background(Palette.panel, RoundedCornerShape(10.dp))
            .border(1.dp, border, RoundedCornerShape(10.dp))
            .clickable(enabled = !locating, onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Outlined.MyLocation, contentDescription = "定位",
            tint = Palette.primary, modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            Text(
                if (locating) "正在获取位置…" else "使用当前位置",
                fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink,
            )
            if (hint.isNotBlank()) {
                Text(hint, fontSize = 11.5.sp, color = Palette.muted,
                    modifier = Modifier.padding(top = 2.dp))
            }
        }
        if (locating) Text("…", fontSize = 14.sp, color = Palette.primary)
    }
}