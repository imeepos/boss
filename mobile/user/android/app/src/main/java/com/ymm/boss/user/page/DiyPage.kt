package com.ymm.boss.user.page

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.Icon
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
import com.ymm.boss.user.api.ServiceApi
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.TopBar
import org.json.JSONObject

/** 对应草稿 docs/user/diy.html:自助排障引导。 */
@Composable
fun DiyScreen(nav: Nav) {
    var sections by remember { mutableStateOf(emptyList<JSONObject>()) }
    var failed by remember { mutableStateOf(false) }

    LaunchedEffect(nav.refreshTick) {
        try { sections = ServiceApi.diySteps().optJSONArray("items").toList() } catch (e: Exception) { failed = true }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("自助排障", onBack = { nav.pop() }, action = "在线客服", onAction = { nav.push(Route.Service) })
        AppCard {
            CardTitle("请选择故障现象")
            sections.forEach { s ->
                CellRow(title = s.optString("title"), desc = s.optString("desc"), right = {
                    Icon(
                        Icons.AutoMirrored.Filled.KeyboardArrowRight,
                        contentDescription = "查看引导",
                        tint = Palette.subtle,
                        modifier = Modifier.size(20.dp),
                    )
                })
            }
            if (sections.isEmpty() && !failed) Notice("加载中…")
            if (failed) Notice("排障引导加载失败，请稍后重试")
        }
        sections.forEach { s -> SectionCard(s, nav) }
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun SectionCard(section: JSONObject, nav: Nav) {
    AppCard {
        CardTitle("${section.optString("title")} · 引导排查")
        section.optJSONArray("steps").toList().forEach { st ->
            CellRow(title = st.optString("title"), desc = st.optString("desc"))
        }
        Text(
            "仍未解决 · 转人工报修",
            fontSize = 13.sp, color = Palette.primary,
            modifier = Modifier.padding(top = 12.dp).clickable { nav.push(Route.Fault) },
        )
    }
}
