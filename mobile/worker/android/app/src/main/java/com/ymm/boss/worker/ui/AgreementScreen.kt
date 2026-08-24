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
import androidx.compose.ui.res.stringArrayResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.ui.theme.Ink
import com.ymm.boss.worker.ui.theme.Muted

// 服务协议全文页:登录/入驻勾选行《服务协议》跳转入口。
// 正文入三语资源(string-array agreement_titles/agreement_bodies),本页只做渲染。
@Composable
fun AgreementScreen(nav: NavHost) {
    val titles = stringArrayResource(R.array.agreement_titles)
    val bodies = stringArrayResource(R.array.agreement_bodies)
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.auth_agreement), onBack = { nav.pop() })
        Column(Modifier.fillMaxWidth().padding(16.dp)) {
            Text(stringResource(R.string.auth_agreement), fontSize = 18.sp, fontWeight = FontWeight.Bold, color = Ink)
            Text(stringResource(R.string.agreement_updated, "2026-08-28"), fontSize = 12.sp, color = Muted,
                modifier = Modifier.padding(top = 6.dp, bottom = 12.dp))
            titles.indices.forEach { i ->
                Text(titles[i], fontSize = 15.sp, fontWeight = FontWeight.W600, color = Ink,
                    modifier = Modifier.padding(top = 12.dp, bottom = 4.dp))
                Text(bodies[i], fontSize = 14.sp, color = Ink, lineHeight = 22.sp)
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}
