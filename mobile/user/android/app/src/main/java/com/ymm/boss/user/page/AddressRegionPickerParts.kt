package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.text
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Palette
import org.json.JSONObject

// AddressRegionPicker 的头部/面包屑/节点行叶组件;拆出守 300 行红线,零行为变更。

@Composable
internal fun PickerHeader(chain: List<JSONObject>, onBack: (Boolean) -> Unit, onJumpTo: (Int) -> Unit) {
    val title = when (chain.size) {
        0 -> "选择大区 / 市"
        1 -> "选择城市 / 市镇"
        2 -> "选择街道 (Barangay)"
        else -> "选择小区 / 楼栋"
    }
    Row(
        Modifier.fillMaxWidth().heightIn(min = 48.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (chain.isNotEmpty()) {
            // 返回箭头触控区扩到 48dp:20dp 裸图标真机点中率差。
            Box(
                Modifier.size(48.dp).clickable(onClick = { onBack(true) }),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回上一级",
                    tint = Palette.muted, modifier = Modifier.size(20.dp),
                )
            }
            Spacer(Modifier.width(8.dp))
        }
        Text(title, fontSize = 16.sp, fontWeight = FontWeight.W600, color = Palette.ink)
    }
    // 面包屑 chips:点任意已选层级直接跳回该层,免逐层按返回。
    if (chain.isNotEmpty()) {
        Row(
            Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            chain.forEachIndexed { index, node ->
                ChainChip(node = node, active = index == chain.size - 1, onClick = { onJumpTo(index) })
            }
        }
        Spacer(Modifier.height(4.dp))
    }
}

/** 面包屑 chip:高≥48dp 保证触控区;active=末位当前层(点击为 no-op)。 */
@Composable
internal fun ChainChip(node: JSONObject, active: Boolean, onClick: () -> Unit) {
    val name = node.optString("name")
    Box(
        Modifier
            .background(
                if (active) Palette.primary else Palette.primary.copy(alpha = 0.08f),
                RoundedCornerShape(12.dp),
            )
            .clickable(onClick = onClick, onClickLabel = "跳回$name")
            // chip 文本提到可点击节点自身:Compose 文本子节点在 a11y 树单独暴露
            // (clickable=false 纯文本行),QA 据 dump 误判 chip 不可点;隐藏子节点
            // 文本、把 text 放到 chip 节点上,uiautomator 可见 text+clickable 同行。
            .semantics { text = AnnotatedString(name) }
            .padding(horizontal = 12.dp)
            .heightIn(min = 48.dp),
        contentAlignment = Alignment.CenterStart,
    ) {
        Text(
            name, fontSize = 12.sp, maxLines = 1,
            color = if (active) Palette.panel else Palette.primary,
            modifier = Modifier.clearAndSetSemantics { },
        )
    }
}

@Composable
internal fun NodeRow(node: JSONObject, enabled: Boolean, onClick: () -> Unit) {
    Row(
        // loading 禁点:同一节点双击会连续入栈两次,链错层。
        Modifier.fillMaxWidth().clickable(enabled = enabled, onClick = onClick).padding(vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Outlined.LocationOn, contentDescription = null,
            tint = Palette.primary.copy(alpha = 0.7f), modifier = Modifier.size(16.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(node.optString("name"), fontSize = 14.sp, color = Palette.ink, modifier = Modifier.weight(1f))
        if (node.optBoolean("hasChildren")) {
            Text("›", fontSize = 16.sp, color = Palette.subtle)
        }
    }
}
