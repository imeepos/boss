package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.json.JSONObject

// Screen3 审核中 + Screen4 状态结果(已认证/驳回 互斥,按 verifyStatus 条件渲染)。

@Composable
internal fun RNReviewingPage(data: JSONObject?) {
    StatusHero(RN.primary, Icons.Filled.Schedule, "审核中", "预计 1-2 个工作日完成审核，结果将以短信通知")
    SummaryCard(data)
    TimelineCard(data)
}

@Composable
internal fun RNApprovedPage(data: JSONObject?) {
    StatusHero(RN.success, Icons.Filled.CheckCircle, "已认证", "实名认证已通过 · ${data?.optString("submitTime").orEmpty().take(19).replace("T", " ")}")
    SummaryCard(data)
    RNSharedCard {
        RNFootnote("认证信息已加密存储；办理入网、停机复机等业务时无需重复认证。")
    }
}

@Composable
internal fun RNRejectedPage(data: JSONObject?, onResubmit: () -> Unit) {
    StatusHero(RN.warn, Icons.Filled.Warning, "未通过认证",
        data?.optString("rejectReason").orEmpty().ifBlank { "提交的信息未通过核验" },
        bg = RN.warnBg)
    SummaryCard(data)
    RNSharedCard {
        RNFootnote("驳回时间：${data?.optString("submitTime").orEmpty().take(19).replace("T", " ")}")
        RNFootnote("请核对姓名、身份证号与证件照片后重新提交。")
    }
    RNWarnButton("重新提交", onClick = onResubmit)
}

/** 状态大卡:图标+标题+副文案,success/warning 底色(spec 状态卡规格)。 */
@Composable
private fun StatusHero(tint: Color, icon: androidx.compose.ui.graphics.vector.ImageVector,
                       title: String, sub: String, bg: Color = Color(0xFFEFFFF4)) {
    Box(
        Modifier.fillMaxWidth().padding(vertical = 4.dp)
            .background(bg, androidx.compose.foundation.shape.RoundedCornerShape(10.dp))
            .padding(vertical = 20.dp),
    ) {
        Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(44.dp))
            Spacer(Modifier.height(8.dp))
            Text(title, fontSize = 17.sp, fontWeight = FontWeight.Bold, color = tint)
            if (sub.isNotBlank()) Text(sub, fontSize = 12.sp, color = RN.muted,
                modifier = Modifier.padding(top = 4.dp, start = 16.dp, end = 16.dp))
        }
    }
}

/** 认证信息摘要:姓名/证件号/手机号均脱敏 + 提交时间(spec: 脱敏号码 tabular-nums 由系统等宽数字近似)。 */
@Composable
private fun SummaryCard(data: JSONObject?) {
    val d = data ?: JSONObject()
    Column(Modifier.fillMaxWidth().padding(top = 8.dp)) {
        CardHead("认证信息")
        RNSharedCard {
            SummaryRow("姓名", d.optString("nameMasked").ifBlank { "—" })
            SummaryRow("证件号码", d.optString("idNoMasked").ifBlank { "—" })
            SummaryRow("手机号", d.optString("phoneMasked").ifBlank { "—" })
            SummaryRow("提交时间", d.optString("submitTime").take(19).replace("T", " ").ifBlank { "—"})
        }
    }
}

@Composable
private fun SummaryRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
        Text(label, fontSize = 14.sp, color = RN.muted)
        Spacer(Modifier.weight(1f))
        Text(value, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = RN.ink)
    }
}

/** 流程时间线:已提交(蓝实心) → 审核中(蓝高亮) → 认证完成(灰描边)。 */
@Composable
private fun TimelineCard(data: JSONObject?) {
    Column(Modifier.fillMaxWidth().padding(top = 8.dp)) {
        CardHead("流程进度")
        RNSharedCard {
            TimelineNode("已提交", "提交时间 " + data?.optString("submitTime").orEmpty().take(19).replace("T", " "),
                state = 0)
            TimelineNode("审核中", "预计 1-2 个工作日", state = 1)
            TimelineNode("认证完成", "审核通过后完成", state = 2)
        }
    }
}

@Composable
private fun TimelineNode(title: String, sub: String, state: Int) {
    Row(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            when (state) {
                0 -> Box(Modifier.size(12.dp).background(RN.primary, CircleShape))
                1 -> Box(Modifier.size(12.dp).background(RN.primary.copy(alpha = 0.25f), CircleShape)
                    .border(2.dp, RN.primary, CircleShape))
                else -> Box(Modifier.size(12.dp).border(1.5.dp, RN.placeholder, CircleShape))
            }
            if (state < 2) Box(Modifier.padding(top = 2.dp).width(2.dp).height(22.dp)
                .background(RN.line))
        }
        Spacer(Modifier.size(12.dp))
        Column {
            Text(title, fontSize = 14.sp, fontWeight = if (state <= 1) FontWeight.W600 else FontWeight.Normal,
                color = if (state <= 1) RN.ink else RN.muted)
            Text(sub, fontSize = 12.sp, color = RN.muted)
        }
    }
}
