package com.ymm.boss.user.page

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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.PrimaryButton
import com.ymm.boss.user.ui.TopBar
import org.json.JSONArray

// 对应草稿 docs/user/agreement.html:用户协议与隐私政策
// 端点: GET /agreement -> {userAgreement: string[], privacyPolicy: string[]}
@Composable
fun AgreementScreen(nav: Nav) {
    var userAgreement by remember { mutableStateOf<List<String>>(emptyList()) }
    var privacyPolicy by remember { mutableStateOf<List<String>>(emptyList()) }
    var failed by remember { mutableStateOf(false) }

    LaunchedEffect(nav.refreshTick) {
        try {
            val d = AccountApi.agreement()
            userAgreement = toStringList(d.optJSONArray("userAgreement"))
            privacyPolicy = toStringList(d.optJSONArray("privacyPolicy"))
        } catch (e: Exception) { failed = true } // 拉取失败时展示兜底文案,不白屏
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("用户协议与隐私政策", onBack = { nav.pop() })
        AgreementCard("用户协议", if (failed) FALLBACK_AGREEMENT else userAgreement)
        AgreementCard("隐私政策", if (failed) FALLBACK_PRIVACY else privacyPolicy)
        PrimaryButton(
            text = "返回登录",
            modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 6.dp),
        ) { nav.pop() }
        Spacer(Modifier.height(14.dp))
    }
}

private fun toStringList(arr: JSONArray?): List<String> {
    if (arr == null) return emptyList()
    val out = ArrayList<String>(arr.length())
    for (i in 0 until arr.length()) out.add(arr.optString(i))
    return out
}

@Composable
private fun AgreementCard(title: String, paras: List<String>) {
    AppCard {
        CardTitle(title)
        if (paras.isEmpty()) {
            EmptyState("暂无内容")
            return@AppCard
        }
        paras.forEachIndexed { i, text ->
            Notice(
                text = "${i + 1}. $text",
                color = Palette.ink.copy(alpha = 0.75f),
            )
        }
    }
}

// 兜底文案:草稿 agreement.html 正文由接口注入,html 本身未内嵌条款文本,
// 此处仅提供最简占位,真实条款以 /agreement 端点返回为准
private val FALLBACK_AGREEMENT = listOf(
    "本协议是您与本平台之间关于宽带办理、账单缴费、故障报修等自助服务事宜的约定。",
    "您注册即视为同意本协议全部条款,并承诺提供真实、准确的实名信息。",
    "平台有权依据法律法规及服务需要调整服务内容,调整将以公告形式通知。",
)
private val FALLBACK_PRIVACY = listOf(
    "平台仅出于提供通信服务之目的收集您的个人信息,包括姓名、证件号、联系方式与装机地址。",
    "未经您的授权,平台不会向任何第三方披露您的个人信息,法律法规另有规定的除外。",
    "您可以在应用内查询、更正个人信息,或联系客服申请注销账号。",
)
