package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted

// 服务协议全文页:登录/入驻勾选行《服务协议》跳转入口。
// 正文以 (标题, 段落) 列表驱动,后续多语言时替换为资源引用即可。
private data class Clause(val title: String, val body: String)

private val CLAUSES = listOf(
    Clause("一、服务内容", "本平台为装维师傅提供工单接收、现场作业、扫码绑定、拍照上报、激活签收等全流程协作服务。师傅应按平台派发的工单信息,在约定时间内完成上门作业并如实上报。"),
    Clause("二、账号与实名", "入驻需提交真实姓名、手机号与身份证号,经平台审核通过后方可接单。账号仅限本人使用,不得转借、出租或共享给第三方。"),
    Clause("三、作业规范", "师傅应遵守安全作业规程,按规定着装并佩戴劳保用品;作业过程中如遇异常(设备缺失、用户拒装、地址错误等)应第一时间通过 App 上报,不得私自关闭或转卖用户设备。"),
    Clause("四、用户信息保护", "师傅在作业中接触的用户姓名、地址、电话等信息均属平台保密信息,仅限本次工单作业使用,不得留存、传播或用于任何其他目的。"),
    Clause("五、费用与结算", "作业费用按平台公示的计费规则结算,收费需通过 App 收费功能线上完成,不得私自向用户收取协议外费用。"),
    Clause("六、违约处理", "存在虚假上报、私收费用、泄露用户信息、恶意投诉用户等行为的,平台有权暂停或终止合作,并保留追偿权利。"),
    Clause("七、协议变更", "平台可根据业务发展修订本协议,修订后将通过 App 公告通知;继续使用服务视为接受修订后的协议。"),
)

@Composable
fun AgreementScreen(nav: NavHost) {
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("服务协议", onBack = { nav.pop() })
        Column(Modifier.fillMaxWidth().padding(16.dp)) {
            Text("装维师傅服务协议", fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Ink)
            Text("更新日期:2026-08-28", fontSize = 12.sp, color = Muted,
                modifier = Modifier.padding(top = 6.dp, bottom = 12.dp))
            CLAUSES.forEach { c ->
                Text(c.title, fontSize = 15.sp, fontWeight = FontWeight.W600, color = Ink,
                    modifier = Modifier.padding(top = 12.dp, bottom = 4.dp))
                Text(c.body, fontSize = 14.sp, color = Ink, lineHeight = 22.sp)
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}
