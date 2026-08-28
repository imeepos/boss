package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ArrowDropDown
import androidx.compose.material.icons.outlined.Place
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.Palette

/** 所在区域行:未选=灰字"点击选择";已选=面包屑 + 已绑定 path 标记;
 *  回显反查失败分两类:HTTP/网络异常=裸段 code+重试(可重试);
 *  响应成功但 path 落在 missing[]=确定性失配(旧格式标签新树无此节点),裸段+灰字说明,不给重试;
 *  403=服务授权异常明示(不静默)。 */
@Composable
internal fun RegionRow(
    breadcrumb: String,
    addressPath: String,
    lookupFailed: Boolean,
    lookupStale: Boolean,
    lookupForbidden: Boolean,
    onRetry: () -> Unit,
    onClick: () -> Unit,
) {
    Row(
        Modifier.fillMaxWidth()
            .background(Palette.bg, RoundedCornerShape(10.dp))
            .border(1.dp, if (addressPath.isNotBlank()) Palette.primary else Palette.line, RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Outlined.Place, contentDescription = null,
            tint = if (addressPath.isNotBlank()) Palette.primary else Palette.muted,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            when {
                lookupForbidden -> {
                    Text("服务授权异常，所在区域暂无法回显", fontSize = 13.sp, color = Palette.err, maxLines = 2)
                    RetryLink(onRetry)
                }
                lookupStale && addressPath.isNotBlank() -> {
                    Text(addressPath.trim().substringAfterLast('.'), fontSize = 13.sp, color = Palette.muted, maxLines = 1)
                    // 确定性失配:lookup 响应成功但 path 在 missing[],重试永远失败,只给说明不给重试。
                    Text("区域编码为旧格式", fontSize = 11.sp, color = Palette.muted,
                        modifier = Modifier.padding(top = 2.dp))
                }
                lookupFailed && addressPath.isNotBlank() -> {
                    Text(addressPath.trim().substringAfterLast('.'), fontSize = 13.sp, color = Palette.muted, maxLines = 1)
                    RetryLink(onRetry)
                }
                breadcrumb.isBlank() -> Text("大区 / 市 / 街道（Barangay）", fontSize = 13.sp, color = Palette.subtle)
                else -> {
                    Text(breadcrumb, fontSize = 13.sp, color = Palette.ink, maxLines = 2)
                    if (addressPath.isNotBlank()) {
                        Text("已绑定层级路径", fontSize = 11.sp, color = Palette.primary,
                            modifier = Modifier.padding(top = 2.dp))
                    }
                }
            }
        }
        Icon(Icons.Outlined.ArrowDropDown, contentDescription = "选择",
            tint = Palette.muted, modifier = Modifier.size(20.dp))
    }
    Spacer(Modifier.height(8.dp))
}

/** 回显反查失败的重试入口:重新触发 lookup(effect 以 lookupTick 为 key)。 */
@Composable
private fun RetryLink(onRetry: () -> Unit) {
    Text(
        "点此重试", fontSize = 12.sp, color = Palette.primary,
        modifier = Modifier.padding(top = 2.dp).clickable(onClick = onRetry),
    )
}
