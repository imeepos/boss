package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.ScanApi
import kotlinx.coroutines.launch
import org.json.JSONArray

// 拍照取证(对齐 docs/worker/photo.html):照片列表 + 现场取证上传
@Composable
fun PhotoScreen(nav: NavHost, no: String) {
    var refresh by remember { mutableStateOf(0) }
    val state by loadOnce(no, refresh) { ScanApi.photos(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("拍照取证", onBack = { nav.pop() })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice("加载失败：${s.message}", red = true) }
            is Load.Ok -> {
                val items = s.data.optJSONArray("items") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    KvRow("取证场景", "现场环境 / 标签污损")
                    SectionTitle("已上传", more = "${items.length()} 张")
                    if (items.length() == 0) Empty("暂无照片")
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        KvRow(it0.optString("fileName"), if (it0.optBoolean("linked")) "已关联" else "待关联")
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    PrimaryButton("拍照上传", modifier = Modifier.fillMaxWidth()) {
                        scope.launch {
                            try {
                                ScanApi.uploadPhoto(no, "现场取证")
                                toast(ctx, "上传成功")
                                refresh++
                            } catch (e: Exception) { toast(ctx, "上传失败：${e.message}") }
                        }
                    }
                    Notice("拍照并上传，自动关联当前工单；弱网自动缓存上传。")
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}