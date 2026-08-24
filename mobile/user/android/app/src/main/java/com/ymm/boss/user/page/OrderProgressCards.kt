package com.ymm.boss.user.page

import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.Palette
import org.json.JSONObject

// 订单详情的进度类卡片:进度概览 stepper、预计上门条、已取消说明卡。

@Composable
internal fun MilestoneBlock(order: JSONObject?) {
    AppCard {
        CardTitle("进度概览", "${order?.optInt("completedStage") ?: 0}/12 已完成")
        Spacer(Modifier.height(8.dp))
        OrderStepper(milestoneOf(order?.optInt("stage") ?: 1))
    }
}

@Composable
internal fun EstimateBanner(order: JSONObject?) {
    val estimate = order?.optString("estimateFinish").orEmpty()
    if (estimate.isBlank()) return
    AppCard(outer = PaddingValues(horizontal = 14.dp, vertical = 6.dp)) {
        Row(
            Modifier.fillMaxWidth().background(Palette.primary.copy(alpha = 0.08f), RoundedCornerShape(8.dp))
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Filled.Schedule, contentDescription = null, tint = Palette.primary, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Column(Modifier.weight(1f)) {
                Text("预计上门:$estimate", fontSize = 13.sp, fontWeight = FontWeight.W500, color = Palette.ink)
                Text("如有变动,师傅将提前电话联系您", fontSize = 11.5.sp, color = Palette.muted, modifier = Modifier.padding(top = 2.dp))
            }
            Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = Palette.subtle, modifier = Modifier.size(18.dp))
        }
    }
}

/**
 * 已取消订单信息卡:根据取消时所处的环节(1~12),给出阶段性的常见原因说明;
 * 配合后端尚未提供 cancelled_at 字段的现状,文案为基于阶段位置的启发式描述。
 */
@Composable
internal fun CancelInfoCard(order: JSONObject?) {
    val stage = order?.optInt("stage", 1)?.coerceIn(1, 12) ?: 1
    val stageLabel = order?.optString("stageLabel").orDefault("")
    val desc = cancelStageDesc(stage)
    AppCard {
        Row(verticalAlignment = Alignment.Top) {
            Box(
                Modifier.size(36.dp).background(Palette.warn.copy(alpha = 0.12f), RoundedCornerShape(10.dp)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Filled.Info, contentDescription = null, tint = Palette.warn, modifier = Modifier.size(20.dp))
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text("取消说明", fontSize = 15.sp, fontWeight = FontWeight.W600, color = Palette.ink)
                Text(
                    desc, fontSize = 13.sp, color = Palette.muted,
                    modifier = Modifier.padding(top = 4.dp),
                )
                if (stageLabel.isNotBlank()) {
                    Text(
                        "取消于:$stageLabel(环节 $stage)",
                        fontSize = 12.sp, color = Palette.subtle,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
        }
    }
}

internal fun cancelStageDesc(stage: Int): String = when {
    stage <= 1 -> "订单创建后立即撤销,通常为重复下单或暂不需要"
    stage <= 3 -> "资源核查/端口预占阶段被取消,该地址可能暂无可用资源"
    stage <= 6 -> "合同收费环节被取消,可能为支付未完成或主动撤销"
    stage <= 9 -> "已派单后被取消,可能为装维条件不具备或用户主动撤销"
    stage <= 11 -> "上门安装过程中被取消,可能为现场条件不满足"
    else -> "订单完成后异常取消,请联系客服核查"
}
