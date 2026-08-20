package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import org.json.JSONArray
import org.json.JSONObject

// 对应草稿 docs/user/verify.html:实名认证状态与核验记录
// 端点: GET /auth/verify -> VerifyStatus{status,nameMasked,idNoMasked,verifyAt,records[]}
@Composable
fun VerifyScreen(nav: Nav) {
    var data by remember { mutableStateOf<JSONObject?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(nav.refreshTick) {
        try { data = AccountApi.verifyStatus() }
        catch (e: Exception) { data = null } // 拉取失败不白屏,按待实名骨架展示
        finally { loading = false }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("实名认证", onBack = { nav.pop() }, action = "帮助", onAction = { nav.push(Route.Help) })
        VerifyBody(loading, data)
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun VerifyBody(loading: Boolean, data: JSONObject?) {
    val d = data ?: JSONObject()
    StatusCard(loading, d)
    StepsCard(d.optString("status", "PENDING"))
    IdInfoCard(d.optString("nameMasked"), d.optString("idNoMasked"))
    RecordsCard(toRecordList(d.optJSONArray("records")))
    ReverifyCard(d.optString("status", "PENDING"))
}

private fun toRecordList(arr: JSONArray?): List<JSONObject> {
    if (arr == null) return emptyList()
    val out = ArrayList<JSONObject>(arr.length())
    for (i in 0 until arr.length()) arr.optJSONObject(i)?.let { out.add(it) }
    return out
}

@Composable
private fun StatusCard(loading: Boolean, d: JSONObject) {
    val verified = d.optString("status") == "VERIFIED"
    AppCard(Modifier.fillMaxWidth()) {
        Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            when {
                loading -> Text("加载中…", fontSize = 13.sp, color = Palette.muted)
                verified -> {
                    Tag("已实名", Palette.success)
                    Text(d.optString("nameMasked"), fontSize = 15.sp, fontWeight = FontWeight.Bold,
                        color = Palette.ink, modifier = Modifier.padding(top = 8.dp))
                    Text("身份证 ${d.optString("idNoMasked")} · ${d.optString("verifyAt")} 核验通过",
                        fontSize = 12.sp, color = Palette.muted)
                }
                else -> {
                    Tag("待实名", Palette.warn)
                    Text(d.optString("nameMasked").ifBlank { "未实名" }, fontSize = 15.sp,
                        fontWeight = FontWeight.Bold, color = Palette.ink, modifier = Modifier.padding(top = 8.dp))
                    Text("办理入网等业务前请先完成实名核验", fontSize = 12.sp, color = Palette.muted)
                }
            }
        }
    }
}

@Composable
private fun StepsCard(status: String) {
    AppCard {
        CardTitle("核验流程")
        val done = status == "VERIFIED"
        Row(Modifier.fillMaxWidth().padding(top = 10.dp, bottom = 12.dp)) {
            listOf("填写信息", "证件上传", "人像比对", "审核通过").forEach { label ->
                StepItem(label, done, Modifier.weight(1f))
            }
        }
        Notice("办理入网、停机复机等业务须完成实名核验;核验方式为「证件 + 人像比对」。证件信息一经核验不可在线变更,证件过期换证需重新核验。")
    }
}

@Composable
private fun StepItem(label: String, done: Boolean, modifier: Modifier = Modifier) {
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Box(Modifier.size(14.dp).background(if (done) Palette.success else Palette.line, CircleShape))
        Text(label, fontSize = 12.sp, color = if (done) Palette.ink else Palette.muted,
            fontWeight = if (done) FontWeight.W600 else FontWeight.Normal,
            modifier = Modifier.padding(top = 6.dp))
    }
}

@Composable
private fun IdInfoCard(name: String, idNo: String) {
    AppCard {
        CardTitle("证件信息")
        OutlinedTextField(value = "身份证", onValueChange = {}, readOnly = true, label = { Text("证件类型") },
            modifier = Modifier.fillMaxWidth().padding(top = 10.dp))
        OutlinedTextField(value = name.ifBlank { "—" }, onValueChange = {}, readOnly = true, label = { Text("姓名") },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
        OutlinedTextField(value = idNo.ifBlank { "—" }, onValueChange = {}, readOnly = true, label = { Text("证件号码") },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
        Text("证件照片(正反面)", fontSize = 13.sp, color = Palette.muted, modifier = Modifier.padding(top = 12.dp, bottom = 6.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            UploadBox("正面已上传", Modifier.weight(1f))
            UploadBox("反面已上传", Modifier.weight(1f))
        }
    }
}

@Composable
private fun UploadBox(text: String, modifier: Modifier = Modifier) {
    Box(
        modifier.height(88.dp).background(Color(0xFFFAFAFA), RoundedCornerShape(8.dp)),
        contentAlignment = Alignment.Center,
    ) { Text(text, fontSize = 12.sp, color = Palette.muted) }
}

@Composable
private fun RecordsCard(records: List<JSONObject>) {
    AppCard {
        CardTitle("核验记录")
        if (records.isEmpty()) EmptyState("暂无核验记录")
        records.forEach { r ->
            val pass = r.optString("result") == "PASS"
            Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(r.optString("method"), fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink)
                    Text(r.optString("time"), fontSize = 12.sp, color = Palette.muted)
                }
                Tag(if (pass) "通过" else "不通过", if (pass) Palette.success else Palette.err)
            }
        }
    }
}

// 草稿此处为 alert 提示文案;POST /auth/verify 需完整证件信息而端侧仅存脱敏字段,
// 故按草稿仅展示重新核验说明,不发起提交
@Composable
private fun ReverifyCard(status: String) {
    var hint by remember { mutableStateOf(false) }
    AppCard {
        Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            if (hint) Notice("重新核验需上传新证件并人像比对,提交后 1 个工作日内审核。", Palette.warn)
            else Notice("证件已过期或信息需变更?", Palette.muted)
            Button(
                onClick = { hint = true },
                enabled = status == "VERIFIED",
                colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
                modifier = Modifier.fillMaxWidth().height(42.dp),
            ) { Text("重新核验") }
        }
    }
}
