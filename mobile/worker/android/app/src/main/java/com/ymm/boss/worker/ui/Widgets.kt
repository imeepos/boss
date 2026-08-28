package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Bg
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Primary2
import com.ymm.boss.worker.ui.theme.Success
import com.ymm.boss.worker.ui.theme.TagColor
import com.ymm.boss.worker.ui.theme.Warn
import com.ymm.boss.worker.ui.theme.tagColor

// 顶栏(对齐 .topbar:渐变底 + 返回 + 标题 + 右侧动作)
@Composable
fun TopBar(title: String, onBack: (() -> Unit)? = null, action: String? = null, onAction: (() -> Unit)? = null) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Brush.linearGradient(listOf(Primary, Primary2)))
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (onBack != null) {
            Text("<", color = Color.White, fontSize = 16.sp,
                modifier = Modifier.clickable { onBack() }.padding(4.dp))
        }
        Text(title, color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.SemiBold,
            modifier = Modifier.weight(1f))
        if (action != null) {
            Text(action, color = Color.White, fontSize = 13.sp,
                modifier = Modifier.clickable { onAction?.invoke() })
        }
    }
}

// 白底圆角卡(对齐 .card)
@Composable
fun Card(modifier: Modifier = Modifier, content: @Composable () -> Unit) {
    Box(
        modifier = modifier
            .fillMaxWidth()
            .background(Color.White, RoundedCornerShape(12.dp))
            .padding(16.dp),
    ) { Column { content() } }
}

@Composable
fun SectionTitle(title: String, more: String? = null, onMore: (() -> Unit)? = null) {
    Row(modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween) {
        Text(title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
        if (more != null) Text(more, fontSize = 12.sp, color = Muted,
            modifier = Modifier.clickable { onMore?.invoke() })
    }
}

// 列表行(对齐 .cell:主文案 + 副文案 + 右侧)
@Composable
fun Cell(title: String, desc: String? = null, onClick: (() -> Unit)? = null,
         right: (@Composable () -> Unit)? = null) {
    Row(modifier = Modifier.fillMaxWidth()
        .clickable(enabled = onClick != null) { onClick?.invoke() }
        .padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Column(modifier = Modifier.weight(1f)) {
            Text(title, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = Ink)
            if (!desc.isNullOrEmpty()) Text(desc, fontSize = 12.sp, color = Muted)
        }
        right?.invoke()
    }
}

// 状态标签(对齐 .tag)
@Composable
fun StatusTag(label: String, status: String? = null, color: TagColor = tagColor(status)) {
    Text(label, fontSize = 12.sp, color = color.fg,
        modifier = Modifier.background(color.bg, RoundedCornerShape(4.dp)).padding(horizontal = 8.dp, vertical = 1.dp))
}

// 提示块(对齐 .notice)
@Composable
fun Notice(text: String, red: Boolean = false) {
    val fg = if (red) Color(0xFFCF1322) else Color(0xFFD46B08)
    val bg = if (red) Color(0xFFFFF1F0) else Color(0xFFFFF7E6)
    Text(text, fontSize = 12.sp, color = fg, modifier = Modifier
        .fillMaxWidth().background(bg, RoundedCornerShape(10.dp))
        .padding(horizontal = 12.dp, vertical = 10.dp))
}

// 三列统计卡(对齐 .stats-card)
@Composable
fun StatCard(title: String, date: String, values: List<Triple<String, String, Color>>) {
    Box(modifier = Modifier.fillMaxWidth()
        .background(Color.White, RoundedCornerShape(12.dp)).padding(16.dp)) {
        Column {
            Row(Modifier.fillMaxWidth().padding(bottom = 12.dp),
                horizontalArrangement = Arrangement.SpaceBetween) {
                Text(title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Ink)
                Text(date, fontSize = 12.sp, color = Muted)
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
                values.forEachIndexed { i, (v, k, c) ->
                    Column(horizontalAlignment = Alignment.CenterHorizontally,
                        modifier = Modifier.weight(1f)) {
                        Text(v, fontSize = 20.sp, fontWeight = FontWeight.Bold, color = c)
                        Text(k, fontSize = 12.sp, color = Muted)
                    }
                    if (i < values.lastIndex) Spacer(Modifier.width(1.dp).height(36.dp).background(Line))
                }
            }
        }
    }
}

// 加载/错误占位
@Composable
fun Loading() {
    Box(Modifier.fillMaxWidth().padding(32.dp), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(modifier = Modifier.size(28.dp), strokeWidth = 2.dp)
    }
}

@Composable
fun ErrorRetry(message: String, onRetry: () -> Unit) {
    Column(Modifier.fillMaxWidth().padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Text(message, fontSize = 13.sp, color = Muted)
        Spacer(Modifier.height(8.dp))
        Text("点击重试", fontSize = 13.sp, color = Primary,
            modifier = Modifier.clickable { onRetry() }.padding(8.dp))
    }
}

// 蓝点状态行(对齐 .st-line)
@Composable
fun StatusLine(text: String, online: Boolean = true) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Box(Modifier.size(8.dp).background(if (online) Color(0xFFA6E9A0) else Color(0xFFFFA39E), CircleShape))
        Text(text, fontSize = 13.sp, color = Color.White.copy(alpha = 0.92f))
    }
}

@Composable
fun PrimaryButton(text: String, modifier: Modifier = Modifier, enabled: Boolean = true, onClick: () -> Unit) {
    Box(modifier = modifier
        .background(if (enabled) Success else Success.copy(alpha = 0.5f), RoundedCornerShape(10.dp))
        .clickable(enabled = enabled) { onClick() }
        .padding(vertical = 12.dp), contentAlignment = Alignment.Center) {
        Text(text, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = Color.White)
    }
}

// 空状态
@Composable
fun Empty(text: String = "暂无数据") {
    Text(text, fontSize = 13.sp, color = Muted,
        modifier = Modifier.fillMaxWidth().padding(vertical = 32.dp), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
}

// 行内动作按钮(白底蓝边/蓝实底,卡片内快捷操作用)
@Composable
fun ActionBtn(text: String, modifier: Modifier = Modifier, primary: Boolean = false, onClick: () -> Unit) {
    val bg = if (primary) Primary else Color.White
    val fg = if (primary) Color.White else Primary
    Text(text, fontSize = 13.sp, color = fg,
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(bg)
            .clickable { onClick() }
            .padding(vertical = 10.dp),
        textAlign = androidx.compose.ui.text.style.TextAlign.Center)
}

// 轻提示(对齐 H5 草稿的 alert/Toast)
fun toast(ctx: android.content.Context, msg: String) {
    android.widget.Toast.makeText(ctx, msg, android.widget.Toast.LENGTH_SHORT).show()
}

val ScreenBg = Bg
val WarnColor = Warn
val SuccessColor = Success
