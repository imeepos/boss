package com.ymm.boss.worker.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Line
import com.ymm.boss.worker.ui.theme.Muted
import com.ymm.boss.worker.ui.theme.Panel
import com.ymm.boss.worker.ui.theme.Primary
import com.ymm.boss.worker.ui.theme.Success
import org.json.JSONArray
import org.json.JSONObject

/**
 * 工单详情·交互组件:
 * - 类型推断(install/repair 后端字段缺失兜底)
 * - 节点状态 / 节点圆点 / 连接线(横版时间轴 primitives)
 * - 底栏主按钮(蓝实底) / 描边次按钮(白底蓝边)
 */

// 后端 TicketDetail 暂未返 type/typeLabel,前端按 terms.md 12/6 环节推断
internal fun inferTicketType(d: JSONObject): String {
    val total = d.optJSONArray("stages")?.length() ?: 12
    return if (total == 6) "REPAIR" else "INSTALL"
}

// 时间轴节点三态(spec §4.4)
internal enum class NodeState { DONE, DOING, PENDING }

internal fun nodeStateOf(result: String): NodeState = when (result) {
    "DONE" -> NodeState.DONE
    "DOING" -> NodeState.DOING
    else -> NodeState.PENDING
}

// 节点圆点(DONE 绿实+白勾 / DOING 蓝实+数字+晕圈 / PENDING 白底灰描边+灰数字)
// 圆 12dp + 显式 lineHeight=fontSize:避免默认行高 × 1.4 撑爆被裁
@Composable
internal fun NodeDot(state: NodeState, num: Int) {
    when (state) {
        NodeState.DONE -> Box(Modifier.size(14.dp).background(Success, CircleShape),
            contentAlignment = Alignment.Center) {
            Text("✓", color = Panel, fontSize = 10.sp, lineHeight = 10.sp,
                fontWeight = FontWeight.Bold)
        }
        NodeState.DOING -> Box(Modifier.size(20.dp).background(Primary.copy(alpha = 0.2f), CircleShape),
            contentAlignment = Alignment.Center) {
            Box(Modifier.size(14.dp).background(Primary, CircleShape),
                contentAlignment = Alignment.Center) {
                Text("$num", color = Panel, fontSize = 10.sp, lineHeight = 10.sp,
                    fontWeight = FontWeight.Bold)
            }
        }
        NodeState.PENDING -> Box(Modifier.size(14.dp).background(Panel, CircleShape)
            .border(1.dp, Line, CircleShape),
            contentAlignment = Alignment.Center) {
            Text("$num", color = Muted, fontSize = 10.sp, lineHeight = 10.sp)
        }
    }
}

// 时间轴连接线(DONE→DONE 绿,其他段灰;2dp 实线)
@Composable
internal fun TimelineConnector(green: Boolean, modifier: Modifier) {
    val color = if (green) Success else Line
    Box(modifier = modifier.height(2.dp).padding(horizontal = 2.dp).background(color))
}

// 底栏主按钮(蓝实底 Primary)
@Composable
internal fun PrimaryAction(text: String, modifier: Modifier, onClick: () -> Unit) {
    Box(modifier = modifier.background(Primary, RoundedCornerShape(10.dp))
        .clickable { onClick() }.padding(vertical = 12.dp),
        contentAlignment = Alignment.Center) {
        Text(text, color = Panel, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
    }
}

// 底栏描边次按钮(白底蓝边)
@Composable
internal fun OutlinedAction(text: String, modifier: Modifier, onClick: () -> Unit) {
    Box(modifier = modifier.background(Panel, RoundedCornerShape(10.dp))
        .border(1.dp, Primary, RoundedCornerShape(10.dp))
        .clickable { onClick() }.padding(vertical = 12.dp),
        contentAlignment = Alignment.Center) {
        Text(text, color = Primary, fontSize = 13.sp, fontWeight = FontWeight.Medium)
    }
}