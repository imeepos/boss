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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
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
@Composable
internal fun NodeDot(state: NodeState, num: Int) {
    when (state) {
        NodeState.DONE -> Box(Modifier.size(12.dp).background(Color(0xFF0AA847), CircleShape),
            contentAlignment = Alignment.Center) {
            Text("✓", color = Color.White, fontSize = 9.sp, fontWeight = FontWeight.Bold)
        }
        NodeState.DOING -> Box(Modifier.size(18.dp).background(Color(0x33086CF5), CircleShape),
            contentAlignment = Alignment.Center) {
            Box(Modifier.size(12.dp).background(Color(0xFF086CF5), CircleShape),
                contentAlignment = Alignment.Center) {
                Text("$num", color = Color.White, fontSize = 9.sp, fontWeight = FontWeight.Bold)
            }
        }
        NodeState.PENDING -> Box(Modifier.size(12.dp).background(Color.White, CircleShape)
            .border(1.dp, Color(0xFFAEB4BE), CircleShape),
            contentAlignment = Alignment.Center) {
            Text("$num", color = Color(0xFFAEB4BE), fontSize = 9.sp)
        }
    }
}

// 时间轴连接线(DONE→DONE 绿,其他段灰;2dp 实线)
@Composable
internal fun TimelineConnector(green: Boolean, modifier: Modifier) {
    val color = if (green) Color(0xFF0AA847) else Color(0xFFE7EAF0)
    Box(modifier = modifier.height(2.dp).padding(horizontal = 2.dp).background(color))
}

// 底栏主按钮(蓝实底 #086CF5)
@Composable
internal fun PrimaryAction(text: String, modifier: Modifier, onClick: () -> Unit) {
    Box(modifier = modifier.background(Color(0xFF086CF5), RoundedCornerShape(10.dp))
        .clickable { onClick() }.padding(vertical = 12.dp),
        contentAlignment = Alignment.Center) {
        Text(text, color = Color.White, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
    }
}

// 底栏描边次按钮(白底蓝边)
@Composable
internal fun OutlinedAction(text: String, modifier: Modifier, onClick: () -> Unit) {
    Box(modifier = modifier.background(Color.White, RoundedCornerShape(10.dp))
        .border(1.dp, Color(0xFF086CF5), RoundedCornerShape(10.dp))
        .clickable { onClick() }.padding(vertical = 12.dp),
        contentAlignment = Alignment.Center) {
        Text(text, color = Color(0xFF086CF5), fontSize = 13.sp, fontWeight = FontWeight.Medium)
    }
}